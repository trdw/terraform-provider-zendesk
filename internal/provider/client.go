package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand/v2"
	"net/http"
	"strconv"
	"sync"
	"time"
)

const (
	defaultMaxRetries    = 5
	defaultRetryAfterSec = 60

	// Backoff bounds for transient (5xx / network) retries.
	retryBaseDelay = 1 * time.Second
	retryMaxDelay  = 30 * time.Second

	throttleStartUsage = 0.5
	throttleMaxUsage   = 0.9
	throttleMinDelay   = 100 * time.Millisecond
	throttleMaxDelay   = 60 * time.Second
)

// isRetryableStatus reports whether an HTTP status code represents a transient
// error worth retrying. 429 is handled separately because it honors Retry-After.
func isRetryableStatus(code int) bool {
	switch code {
	case http.StatusRequestTimeout, // 408
		http.StatusInternalServerError, // 500
		http.StatusBadGateway,          // 502
		http.StatusServiceUnavailable,  // 503
		http.StatusGatewayTimeout:      // 504
		return true
	}
	// Cloudflare-specific edge errors (520-524): origin unreachable / timed out.
	if code >= 520 && code <= 524 {
		return true
	}
	return false
}

// backoffDelay returns an exponential backoff duration with full jitter for the
// given zero-based attempt number (capped at retryMaxDelay).
func backoffDelay(attempt int) time.Duration {
	delay := float64(retryBaseDelay) * math.Pow(2, float64(attempt))
	if delay > float64(retryMaxDelay) {
		delay = float64(retryMaxDelay)
	}
	// Full jitter with a floor of half the computed delay.
	return time.Duration(delay/2 + rand.Float64()*delay/2)
}

type ZendeskClient struct {
	baseURL    string
	email      string
	apiToken   string
	httpClient *http.Client
	maxRetries int

	mu                 sync.Mutex
	rateLimitTotal     int
	rateLimitRemaining int
	rateLimitResetAt   time.Time
}

type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("Zendesk API error (HTTP %d): %s", e.StatusCode, e.Body)
}

func IsNotFound(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.StatusCode == 404
	}
	return false
}

func NewZendeskClient(subdomain, email, apiToken string) *ZendeskClient {
	return &ZendeskClient{
		baseURL:            fmt.Sprintf("https://%s.zendesk.com", subdomain),
		email:              email,
		apiToken:           apiToken,
		httpClient:         &http.Client{Timeout: 60 * time.Second},
		maxRetries:         defaultMaxRetries,
		rateLimitRemaining: -1,
	}
}

func (c *ZendeskClient) updateRateLimits(resp *http.Response) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if v := resp.Header.Get("X-Rate-Limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.rateLimitTotal = n
		}
	}

	remaining := resp.Header.Get("X-Rate-Limit-Remaining")
	if remaining == "" {
		remaining = resp.Header.Get("ratelimit-remaining")
	}
	if remaining != "" {
		if n, err := strconv.Atoi(remaining); err == nil {
			c.rateLimitRemaining = n
		}
	}

	if v := resp.Header.Get("ratelimit-reset"); v != "" {
		if secs, err := strconv.Atoi(v); err == nil {
			c.rateLimitResetAt = time.Now().Add(time.Duration(secs) * time.Second)
		}
	}
}

func (c *ZendeskClient) throttleIfNeeded() {
	c.mu.Lock()
	remaining := c.rateLimitRemaining
	resetAt := c.rateLimitResetAt
	total := c.rateLimitTotal
	c.mu.Unlock()

	if remaining < 0 || total <= 0 {
		return
	}

	usage := 1.0 - float64(remaining)/float64(total)

	if usage >= throttleMaxUsage {
		wait := time.Until(resetAt)
		if wait > 0 {
			log.Printf("[WARN] Zendesk rate limit critical (%.0f%% used, %d/%d remaining), waiting %s until reset",
				usage*100, remaining, total, wait.Round(time.Second))
			time.Sleep(wait)
		}
		return
	}

	if usage >= throttleStartUsage {
		t := (usage - throttleStartUsage) / (throttleMaxUsage - throttleStartUsage)
		delay := time.Duration(float64(throttleMinDelay) * math.Pow(float64(throttleMaxDelay)/float64(throttleMinDelay), t))
		log.Printf("[DEBUG] Zendesk rate limit throttle (%.0f%% used, %d/%d remaining), delaying %s",
			usage*100, remaining, total, delay.Round(time.Millisecond))
		time.Sleep(delay)
	}
}

func (c *ZendeskClient) doRequest(method, path string, body interface{}, result interface{}) error {
	url := c.baseURL + path

	var jsonBody []byte
	if body != nil {
		var err error
		jsonBody, err = json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshaling request body: %w", err)
		}
	}

	var lastErr error

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		c.throttleIfNeeded()

		var reqBody io.Reader
		if jsonBody != nil {
			reqBody = bytes.NewBuffer(jsonBody)
		}

		req, err := http.NewRequest(method, url, reqBody)
		if err != nil {
			return fmt.Errorf("creating request: %w", err)
		}

		req.SetBasicAuth(c.email+"/token", c.apiToken)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			// Transport-level failure (connection reset, DNS, TLS, timeout).
			// These are transient, so back off and retry.
			lastErr = fmt.Errorf("executing request: %w", err)
			if attempt < c.maxRetries {
				delay := backoffDelay(attempt)
				log.Printf("[WARN] Zendesk request error on %s %s (%v), retry %d/%d after %s",
					method, path, err, attempt+1, c.maxRetries, delay.Round(time.Millisecond))
				time.Sleep(delay)
				continue
			}
			return lastErr
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("reading response body: %w", err)
			if attempt < c.maxRetries {
				delay := backoffDelay(attempt)
				log.Printf("[WARN] Zendesk response read error on %s %s (%v), retry %d/%d after %s",
					method, path, err, attempt+1, c.maxRetries, delay.Round(time.Millisecond))
				time.Sleep(delay)
				continue
			}
			return lastErr
		}

		c.updateRateLimits(resp)

		// 429: respect the server's Retry-After (or fall back to a default).
		if resp.StatusCode == http.StatusTooManyRequests && attempt < c.maxRetries {
			retryAfter := defaultRetryAfterSec
			if v := resp.Header.Get("Retry-After"); v != "" {
				if n, err := strconv.Atoi(v); err == nil && n > 0 {
					retryAfter = n
				}
			}
			log.Printf("[WARN] Zendesk rate limit hit (429 on %s %s), retry %d/%d after %ds",
				method, path, attempt+1, c.maxRetries, retryAfter)
			time.Sleep(time.Duration(retryAfter) * time.Second)
			lastErr = &APIError{StatusCode: resp.StatusCode, Body: string(respBody)}
			continue
		}

		// Transient gateway/server errors (e.g. Cloudflare 502/503/504): back off and retry.
		if isRetryableStatus(resp.StatusCode) && attempt < c.maxRetries {
			lastErr = &APIError{StatusCode: resp.StatusCode, Body: string(respBody)}
			delay := backoffDelay(attempt)
			log.Printf("[WARN] Zendesk transient error (HTTP %d on %s %s), retry %d/%d after %s",
				resp.StatusCode, method, path, attempt+1, c.maxRetries, delay.Round(time.Millisecond))
			time.Sleep(delay)
			continue
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return &APIError{
				StatusCode: resp.StatusCode,
				Body:       string(respBody),
			}
		}

		if result != nil && len(respBody) > 0 {
			if err := json.Unmarshal(respBody, result); err != nil {
				return fmt.Errorf("unmarshaling response: %w", err)
			}
		}

		return nil
	}

	// Retries exhausted; surface the most recent transient error.
	if lastErr != nil {
		return fmt.Errorf("%s %s failed after %d retries: %w", method, path, c.maxRetries, lastErr)
	}
	return &APIError{
		StatusCode: 0,
		Body:       fmt.Sprintf("request failed on %s %s after %d retries", method, path, c.maxRetries),
	}
}

func (c *ZendeskClient) Get(path string, result interface{}) error {
	return c.doRequest(http.MethodGet, path, nil, result)
}

func (c *ZendeskClient) Post(path string, body interface{}, result interface{}) error {
	return c.doRequest(http.MethodPost, path, body, result)
}

func (c *ZendeskClient) Put(path string, body interface{}, result interface{}) error {
	return c.doRequest(http.MethodPut, path, body, result)
}

func (c *ZendeskClient) Patch(path string, body interface{}, result interface{}) error {
	return c.doRequest(http.MethodPatch, path, body, result)
}

func (c *ZendeskClient) Delete(path string) error {
	return c.doRequest(http.MethodDelete, path, nil, nil)
}

// valueToString converts an API value (which may be string, float64, bool, etc.) to a string.
func valueToString(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case float64:
		if val == float64(int64(val)) {
			return strconv.FormatInt(int64(val), 10)
		}
		return strconv.FormatFloat(val, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(val)
	case nil:
		return ""
	default:
		b, _ := json.Marshal(val)
		return string(b)
	}
}

// stringToValue converts a Terraform string value to an appropriate API value.
func stringToValue(s string) interface{} {
	if s == "" {
		return s
	}
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return i
	}
	if b, err := strconv.ParseBool(s); err == nil {
		return b
	}
	var arr []interface{}
	if err := json.Unmarshal([]byte(s), &arr); err == nil {
		return arr
	}
	return s
}
