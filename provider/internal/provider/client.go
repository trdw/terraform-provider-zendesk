package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"
)

const (
	maxRetries           = 3
	defaultRetryAfterSec = 60

	throttleStartUsage = 0.5
	throttleMaxUsage   = 0.9
	throttleMinDelay   = 100 * time.Millisecond
	throttleMaxDelay   = 60 * time.Second
)

type ZendeskClient struct {
	baseURL    string
	email      string
	apiToken   string
	httpClient *http.Client

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

	for attempt := 0; attempt <= maxRetries; attempt++ {
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
			return fmt.Errorf("executing request: %w", err)
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return fmt.Errorf("reading response body: %w", err)
		}

		c.updateRateLimits(resp)

		if resp.StatusCode == 429 && attempt < maxRetries {
			retryAfter := defaultRetryAfterSec
			if v := resp.Header.Get("Retry-After"); v != "" {
				if n, err := strconv.Atoi(v); err == nil && n > 0 {
					retryAfter = n
				}
			}
			log.Printf("[WARN] Zendesk rate limit hit (429 on %s %s), retry %d/%d after %ds",
				method, path, attempt+1, maxRetries, retryAfter)
			time.Sleep(time.Duration(retryAfter) * time.Second)
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

	return &APIError{
		StatusCode: 429,
		Body:       fmt.Sprintf("rate limit exceeded on %s %s after %d retries", method, path, maxRetries),
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
