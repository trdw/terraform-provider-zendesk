#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

RELATED_RESOURCES=""
POSITIONAL=()
for arg in "$@"; do
  case "$arg" in
    --related-resources=*)
      RELATED_RESOURCES="${arg#*=}"
      ;;
    *)
      POSITIONAL+=("$arg")
      ;;
  esac
done

if [[ ${#POSITIONAL[@]} -lt 2 ]]; then
  echo "Usage: $0 <infra-dir> <zendesk-admin-url> [--related-resources=y|n]"
  echo "Example: $0 /path/to/infrastructure/sandbox https://my-organization.zendesk.com/admin/objects-rules/rules/triggers/360226671071"
  exit 1
fi

INFRA_DIR="${POSITIONAL[0]}"
URL="${POSITIONAL[1]}"
GENERATORS_DIR="${SCRIPT_DIR}/generators"

if [[ ! -d "$INFRA_DIR" ]]; then
  echo "Error: infrastructure directory '${INFRA_DIR}' does not exist"
  exit 1
fi

# Extract subdomain and email from provider.tf
PROVIDER_TF="${INFRA_DIR}/provider.tf"
if [[ ! -f "$PROVIDER_TF" ]]; then
  echo "Error: provider.tf not found at ${PROVIDER_TF}"
  exit 1
fi

ZENDESK_SUBDOMAIN=$(grep -oE 'subdomain\s*=\s*"[^"]+"' "$PROVIDER_TF" | grep -oE '"[^"]+"' | tr -d '"')
ZENDESK_EMAIL=$(grep -oE 'email\s*=\s*"[^"]+"' "$PROVIDER_TF" | grep -oE '"[^"]+"' | tr -d '"')

if [[ -z "$ZENDESK_SUBDOMAIN" ]]; then
  echo "Error: could not extract subdomain from ${PROVIDER_TF}"
  exit 1
fi
if [[ -z "$ZENDESK_EMAIL" ]]; then
  echo "Error: could not extract email from ${PROVIDER_TF}"
  exit 1
fi

: "${TF_VAR_zendesk_api_token:?Set TF_VAR_zendesk_api_token}"

echo "==> Using subdomain: ${ZENDESK_SUBDOMAIN}, email: ${ZENDESK_EMAIL}"

BASE_URL="https://${ZENDESK_SUBDOMAIN}.zendesk.com"
AUTH="${ZENDESK_EMAIL}/token:${TF_VAR_zendesk_api_token}"

# ---------------------------------------------------------------------------
# Parse the URL to determine resource type and ID
# ---------------------------------------------------------------------------
# Known URL patterns (hyphen or underscore variants are accepted):
#   /admin/objects-rules/rules/triggers/<id>                 -> trigger
#   /admin/objects-rules/rules/trigger-categories/<id>       -> trigger_category
#   /admin/objects-rules/rules/automations/<id>              -> automation
#   /admin/workspaces/agent-workspace/views/<id>             -> view
#   /admin/objects-rules/rules/macros/<id>                   -> macro
#   /admin/objects-rules/tickets/ticket-forms/edit/<id>      -> ticket_form
#   /admin/objects-rules/tickets/ticket-fields/<id>          -> ticket_field
#   /admin/people/configuration/user-fields/<id>             -> user_field
#   /admin/people/team/groups/<id>                           -> group
#   /admin/people/team/roles/<id>                            -> custom_role
#   /admin/people/team/members/<id>                          -> user
#   /admin/account/brand_management/brands/<id>              -> brand

RESOURCE_ID=$(echo "$URL" | grep -oE '[0-9]+$')
if [[ -z "$RESOURCE_ID" ]]; then
  echo "Error: Could not extract resource ID from URL"
  exit 1
fi

# Determine resource type from URL path.
# Order matters: put more-specific patterns (e.g. trigger-categories) before
# more-general ones (e.g. triggers) so they match first.
if echo "$URL" | grep -qE '/trigger[-_]categor'; then
  RESOURCE_TYPE="trigger_category"
  API_PATH="/api/v2/trigger_categories/${RESOURCE_ID}"
  TF_TYPE="zendesk_trigger_category"
elif echo "$URL" | grep -qE '/triggers/'; then
  RESOURCE_TYPE="trigger"
  API_PATH="/api/v2/triggers/${RESOURCE_ID}"
  TF_TYPE="zendesk_trigger"
elif echo "$URL" | grep -qE '/automations/'; then
  RESOURCE_TYPE="automation"
  API_PATH="/api/v2/automations/${RESOURCE_ID}"
  TF_TYPE="zendesk_automation"
elif echo "$URL" | grep -qE '/views/'; then
  RESOURCE_TYPE="view"
  API_PATH="/api/v2/views/${RESOURCE_ID}"
  TF_TYPE="zendesk_view"
elif echo "$URL" | grep -qE '/macros/'; then
  RESOURCE_TYPE="macro"
  API_PATH="/api/v2/macros/${RESOURCE_ID}"
  TF_TYPE="zendesk_macro"
elif echo "$URL" | grep -qE '/ticket[-_]forms/'; then
  RESOURCE_TYPE="ticket_form"
  API_PATH="/api/v2/ticket_forms/${RESOURCE_ID}"
  TF_TYPE="zendesk_ticket_form"
elif echo "$URL" | grep -qE '/ticket[-_]fields/'; then
  RESOURCE_TYPE="ticket_field"
  API_PATH="/api/v2/ticket_fields/${RESOURCE_ID}"
  TF_TYPE="zendesk_ticket_field"
elif echo "$URL" | grep -qE '/user[-_]fields/'; then
  RESOURCE_TYPE="user_field"
  API_PATH="/api/v2/user_fields/${RESOURCE_ID}"
  TF_TYPE="zendesk_user_field"
elif echo "$URL" | grep -qE '/groups/'; then
  RESOURCE_TYPE="group"
  API_PATH="/api/v2/groups/${RESOURCE_ID}"
  TF_TYPE="zendesk_group"
elif echo "$URL" | grep -qE '/roles/'; then
  RESOURCE_TYPE="custom_role"
  API_PATH="/api/v2/custom_roles/${RESOURCE_ID}"
  TF_TYPE="zendesk_custom_role"
elif echo "$URL" | grep -qE '/members/'; then
  RESOURCE_TYPE="user"
  API_PATH="/api/v2/users/${RESOURCE_ID}"
  TF_TYPE="zendesk_user"
elif echo "$URL" | grep -qE '/brands/'; then
  RESOURCE_TYPE="brand"
  API_PATH="/api/v2/brands/${RESOURCE_ID}"
  TF_TYPE="zendesk_brand"
else
  echo "Error: Could not determine resource type from URL"
  echo "Supported: triggers, trigger-categories, automations, views, macros, ticket-forms, ticket-fields, user-fields, groups, roles, members, brands"
  exit 1
fi

echo "==> Detected resource type: ${RESOURCE_TYPE} (ID: ${RESOURCE_ID})"

# ---------------------------------------------------------------------------
# Fetch the resource JSON from the Zendesk API
# ---------------------------------------------------------------------------
echo "==> Fetching ${RESOURCE_TYPE} from Zendesk API..."
API_RESPONSE=$(curl -s -u "$AUTH" "${BASE_URL}${API_PATH}")

# Check for errors
if echo "$API_RESPONSE" | grep -q '"error"'; then
  echo "Error: API returned an error:"
  echo "$API_RESPONSE" | python3 -m json.tool 2>/dev/null || echo "$API_RESPONSE"
  exit 1
fi

echo "$API_RESPONSE" | python3 -m json.tool

# ---------------------------------------------------------------------------
# Helper: convert a string to snake_case
# ---------------------------------------------------------------------------
to_snake_case() {
  echo "$1" | tr '[:upper:]' '[:lower:]' | sed -E 's/[^a-z0-9]+/_/g; s/^_+//; s/_+$//'
}

# Accepts y, yes, true (any case) — matches the values users typically pass to
# --related-resources or type at the prompt.
is_truthy() {
  case "$1" in
    [Yy]|[Yy][Ee][Ss]|[Tt][Rr][Uu][Ee]) return 0 ;;
    *) return 1 ;;
  esac
}

# Escape a value for use inside an HCL double-quoted string. Generators
# accumulate output in a `tf` buffer that is later passed through `echo -e`,
# so every `\X` we want to appear in the .tf file has to be `\\X` in the
# buffer, which is `\\\\X` as a bash literal — hence the four backslashes.
# Read a boolean from a JSON document, returning the source's explicit
# `true`/`false` when present, otherwise the supplied default. Avoids the
# `... // default` trap in jq where `false` is treated as falsy and replaced.
# Usage: v=$(jq_bool "$json" '.path.to.field' true)
jq_bool() {
  local json="$1" path="$2" default="$3"
  local v
  v=$(echo "$json" | jq -r "$path")
  if [[ "$v" == "true" || "$v" == "false" ]]; then
    printf '%s' "$v"
  else
    printf '%s' "$default"
  fi
}

hcl_escape() {
  local s="$1"
  s="${s//\\/\\\\\\\\}"   # \   -> \\  (HCL backslash escape)
  s="${s//\"/\\\\\"}"     # "   -> \"  (HCL quote escape)
  s="${s//$'\n'/\\\\n}"   # LF  -> \n
  s="${s//$'\r'/\\\\r}"   # CR  -> \r
  s="${s//$'\t'/\\\\t}"   # TAB -> \t
  s="${s//\$\{/\$\${}"    # ${ -> $${  (defuse HCL string interpolation)
  s="${s//%\{/%%{}"       # %{ -> %%{  (defuse HCL template directive)
  printf '%s' "$s"
}

# ---------------------------------------------------------------------------
# Generate the .tf file deterministically from the API JSON
# ---------------------------------------------------------------------------
GENERATOR="${GENERATORS_DIR}/${RESOURCE_TYPE}.sh"
if [[ ! -f "$GENERATOR" ]]; then
  echo "Error: no generator found for resource type '${RESOURCE_TYPE}'"
  echo "Expected: ${GENERATOR}"
  exit 1
fi

source "$GENERATOR"

# ---------------------------------------------------------------------------
# Pre-fetch related resources that need to be reflected in the parent
# resource's generated .tf (e.g. a ticket_form's custom field references).
# Field .tf files are written here so they exist before terraform import runs.
# ---------------------------------------------------------------------------
TICKET_FIELD_IMPORTS=""        # newline-separated "<id> <res_name>" pairs
TICKET_FIELD_MAP="{}"          # JSON object: {"<id>": "<res_name>"}
BRAND_IMPORTS=""               # newline-separated "<id> <res_name>" pairs
BRAND_MAP="{}"                 # JSON object: {"<id>": "<res_name>"}
if [[ "$RESOURCE_TYPE" == "ticket_form" ]]; then
  if [[ -n "$RELATED_RESOURCES" ]]; then
    IMPORT_RELATED="$RELATED_RESOURCES"
  else
    read -rp "==> Also import the form's custom ticket fields and restricted brands? [y/N] " IMPORT_RELATED
  fi
  if is_truthy "$IMPORT_RELATED"; then
    source "${GENERATORS_DIR}/ticket_field.sh"
    source "${GENERATORS_DIR}/brand.sh"

    FIELD_IDS=$(echo "$API_RESPONSE" | jq -r '.ticket_form.ticket_field_ids[]?')
    for FIELD_ID in $FIELD_IDS; do
      echo "==> Fetching ticket field ${FIELD_ID}..."
      FIELD_RESPONSE=$(curl -s -u "$AUTH" "${BASE_URL}/api/v2/ticket_fields/${FIELD_ID}")

      if echo "$FIELD_RESPONSE" | grep -q '"error"'; then
        echo "  Skip: API error for field ${FIELD_ID}"
        continue
      fi

      FIELD_TITLE=$(echo "$FIELD_RESPONSE" | jq -r '.ticket_field.title')
      FIELD_RES_NAME=$(to_snake_case "$FIELD_TITLE")
      FIELD_TF_FILE="${INFRA_DIR}/ticket_field_${FIELD_RES_NAME}.tf"
      REMOVABLE=$(echo "$FIELD_RESPONSE" | jq -r '.ticket_field.removable // false')

      if [[ "$REMOVABLE" == "true" ]]; then
        # Custom field — emit a resource and import it.
        FIELD_TF_CONTENT=$(generate_ticket_field "$FIELD_RESPONSE")
        FIELD_RES_NAME=$(echo "$FIELD_TF_CONTENT" | grep -oE 'resource "zendesk_ticket_field" "[a-z_0-9]+"' | sed -E 's/resource "zendesk_ticket_field" "([a-z_0-9]+)"/\1/')
        FIELD_TF_FILE="${INFRA_DIR}/ticket_field_${FIELD_RES_NAME}.tf"

        if [[ -f "$FIELD_TF_FILE" ]]; then
          echo "  Skip write: ${FIELD_TF_FILE} already exists"
        else
          echo "$FIELD_TF_CONTENT" > "$FIELD_TF_FILE"
          echo "  Wrote ${FIELD_TF_FILE}"
        fi

        TICKET_FIELD_IMPORTS+="${FIELD_ID} ${FIELD_RES_NAME}"$'\n'
        TICKET_FIELD_MAP=$(echo "$TICKET_FIELD_MAP" | jq --arg id "$FIELD_ID" --arg addr "zendesk_ticket_field.${FIELD_RES_NAME}" '. + {($id): $addr}')
      else
        # System field — emit a `data` block (no import needed).
        if [[ -f "$FIELD_TF_FILE" ]]; then
          echo "  Skip write: ${FIELD_TF_FILE} already exists (${FIELD_TITLE}, system)"
        else
          {
            echo "data \"zendesk_ticket_field\" \"${FIELD_RES_NAME}\" {"
            echo "  title = \"$(hcl_escape "$FIELD_TITLE")\""
            echo "}"
          } > "$FIELD_TF_FILE"
          echo "  Wrote ${FIELD_TF_FILE} (${FIELD_TITLE}, system)"
        fi
        TICKET_FIELD_MAP=$(echo "$TICKET_FIELD_MAP" | jq --arg id "$FIELD_ID" --arg addr "data.zendesk_ticket_field.${FIELD_RES_NAME}" '. + {($id): $addr}')
      fi
    done

    BRAND_IDS=$(echo "$API_RESPONSE" | jq -r '.ticket_form.restricted_brand_ids[]?')
    for BRAND_ID in $BRAND_IDS; do
      echo "==> Fetching brand ${BRAND_ID}..."
      BRAND_RESPONSE=$(curl -s -u "$AUTH" "${BASE_URL}/api/v2/brands/${BRAND_ID}")

      if echo "$BRAND_RESPONSE" | grep -q '"error"'; then
        echo "  Skip: API error for brand ${BRAND_ID}"
        continue
      fi

      BRAND_TF_CONTENT=$(generate_brand "$BRAND_RESPONSE")
      BRAND_RES_NAME=$(echo "$BRAND_TF_CONTENT" | grep -oE 'resource "zendesk_brand" "[a-z_0-9]+"' | sed -E 's/resource "zendesk_brand" "([a-z_0-9]+)"/\1/')
      BRAND_TF_FILE="${INFRA_DIR}/brand_${BRAND_RES_NAME}.tf"

      if [[ -f "$BRAND_TF_FILE" ]]; then
        echo "  Skip write: ${BRAND_TF_FILE} already exists"
      else
        echo "$BRAND_TF_CONTENT" > "$BRAND_TF_FILE"
        echo "  Wrote ${BRAND_TF_FILE}"
      fi

      BRAND_IMPORTS+="${BRAND_ID} ${BRAND_RES_NAME}"$'\n'
      BRAND_MAP=$(echo "$BRAND_MAP" | jq --arg id "$BRAND_ID" --arg addr "zendesk_brand.${BRAND_RES_NAME}" '. + {($id): $addr}')
    done
  fi
fi

echo "==> Generating Terraform file..."
if [[ "$RESOURCE_TYPE" == "ticket_form" ]]; then
  TF_CONTENT=$(generate_ticket_form "$API_RESPONSE" "$TICKET_FIELD_MAP" "$BRAND_MAP")
else
  TF_CONTENT=$(generate_${RESOURCE_TYPE} "$API_RESPONSE")
fi

if [[ -z "$TF_CONTENT" ]]; then
  echo "Error: generator returned empty output"
  exit 1
fi

TF_RESOURCE_NAME=$(echo "$TF_CONTENT" | grep -oE "resource \"${TF_TYPE}\" \"([a-z_0-9]+)\"" | sed -E "s/resource \"${TF_TYPE}\" \"([a-z_0-9]+)\"/\1/")
if [[ -z "$TF_RESOURCE_NAME" ]]; then
  echo "Error: Could not extract resource name from generated Terraform"
  echo "$TF_CONTENT"
  exit 1
fi

TF_FILE="${INFRA_DIR}/${RESOURCE_TYPE}_${TF_RESOURCE_NAME}.tf"

echo "$TF_CONTENT" > "$TF_FILE"
echo "==> Written to ${TF_FILE}"
echo ""
echo "$TF_CONTENT"
echo ""

# ---------------------------------------------------------------------------
# Optional: import group members
# ---------------------------------------------------------------------------
MEMBERS_JSON=""
if [[ "$RESOURCE_TYPE" == "group" ]]; then
  if [[ -n "$RELATED_RESOURCES" ]]; then
    IMPORT_MEMBERS="$RELATED_RESOURCES"
  else
    read -rp "==> Also import group members? [y/N] " IMPORT_MEMBERS
  fi
  if is_truthy "$IMPORT_MEMBERS"; then
    echo "==> Fetching group memberships..."
    MEMBERSHIPS_RESPONSE=$(curl -s -u "$AUTH" "${BASE_URL}/api/v2/groups/${RESOURCE_ID}/memberships")

    # Build a JSON array of {user_name, user_id, membership_id} for each member
    MEMBER_USER_IDS=$(echo "$MEMBERSHIPS_RESPONSE" | jq -r '.group_memberships[].user_id')
    MEMBERS_JSON="[]"

    for USER_ID in $MEMBER_USER_IDS; do
      echo "  Fetching user ${USER_ID}..."
      USER_RESPONSE=$(curl -s -u "$AUTH" "${BASE_URL}/api/v2/users/${USER_ID}")
      USER_NAME=$(echo "$USER_RESPONSE" | jq -r '.user.name')
      MEMBERSHIP_ID=$(echo "$MEMBERSHIPS_RESPONSE" | jq -r ".group_memberships[] | select(.user_id == ${USER_ID}) | .id")

      MEMBERS_JSON=$(echo "$MEMBERS_JSON" | jq \
        --arg name "$USER_NAME" \
        --arg uid "$USER_ID" \
        --arg mid "$MEMBERSHIP_ID" \
        '. += [{"user_name": $name, "user_id": $uid, "membership_id": $mid}]')
    done

    echo "==> Found $(echo "$MEMBERS_JSON" | jq 'length') members"

    # Generate and append membership block to the group .tf file
    MEMBERSHIP_CONTENT=$(generate_group_membership "$MEMBERS_JSON" "$TF_RESOURCE_NAME")
    if [[ -n "$MEMBERSHIP_CONTENT" ]]; then
      echo "$MEMBERSHIP_CONTENT" >> "$TF_FILE"
      echo "==> Appended group_membership to ${TF_FILE}"
    fi

    # Generate and append data sources to users.tf
    USERS_TF="${INFRA_DIR}/users.tf"
    DATA_SOURCES=$(generate_user_data_sources "$MEMBERS_JSON" "$USERS_TF")
    if [[ -n "$DATA_SOURCES" ]]; then
      echo "$DATA_SOURCES" >> "$USERS_TF"
      echo "==> Appended user data sources to ${USERS_TF}"
    fi

    echo ""
    echo "=== Generated group file ==="
    cat "$TF_FILE"
    echo ""
  fi
fi

# ---------------------------------------------------------------------------
# Import the resource into Terraform state
# ---------------------------------------------------------------------------
echo "==> Importing ${TF_TYPE}.${TF_RESOURCE_NAME} (ID: ${RESOURCE_ID})..."
(cd "$INFRA_DIR" && terraform import "${TF_TYPE}.${TF_RESOURCE_NAME}" "$RESOURCE_ID")

# ---------------------------------------------------------------------------
# Import ticket form's related resources (custom fields and restricted brands)
# into Terraform state.
# ---------------------------------------------------------------------------
if [[ -n "$TICKET_FIELD_IMPORTS" || -n "$BRAND_IMPORTS" ]]; then
  EXISTING_STATE=$(cd "$INFRA_DIR" && terraform state list 2>/dev/null || true)

  while IFS=' ' read -r FIELD_ID FIELD_RES_NAME; do
    [[ -z "$FIELD_ID" ]] && continue
    IMPORT_ADDR="zendesk_ticket_field.${FIELD_RES_NAME}"
    if echo "$EXISTING_STATE" | grep -qx "$IMPORT_ADDR"; then
      echo "==> Skip import (already in state): ${IMPORT_ADDR}"
      continue
    fi
    echo "==> Importing ${IMPORT_ADDR} (ID: ${FIELD_ID})..."
    (cd "$INFRA_DIR" && terraform import "$IMPORT_ADDR" "$FIELD_ID")
  done <<< "$TICKET_FIELD_IMPORTS"

  while IFS=' ' read -r BRAND_ID BRAND_RES_NAME; do
    [[ -z "$BRAND_ID" ]] && continue
    IMPORT_ADDR="zendesk_brand.${BRAND_RES_NAME}"
    if echo "$EXISTING_STATE" | grep -qx "$IMPORT_ADDR"; then
      echo "==> Skip import (already in state): ${IMPORT_ADDR}"
      continue
    fi
    echo "==> Importing ${IMPORT_ADDR} (ID: ${BRAND_ID})..."
    (cd "$INFRA_DIR" && terraform import "$IMPORT_ADDR" "$BRAND_ID")
  done <<< "$BRAND_IMPORTS"
fi

# ---------------------------------------------------------------------------
# Import group memberships into Terraform state
# ---------------------------------------------------------------------------
if [[ -n "$MEMBERS_JSON" && "$MEMBERS_JSON" != "[]" ]]; then
  MEMBER_COUNT=$(echo "$MEMBERS_JSON" | jq 'length')
  for i in $(seq 0 $((MEMBER_COUNT - 1))); do
    USER_NAME=$(echo "$MEMBERS_JSON" | jq -r ".[$i].user_name")
    MEMBERSHIP_ID=$(echo "$MEMBERS_JSON" | jq -r ".[$i].membership_id")
    DATA_NAME=$(to_snake_case "$USER_NAME")
    IMPORT_ADDR="zendesk_group_membership.${TF_RESOURCE_NAME}[\"$(echo "$MEMBERS_JSON" | jq -r ".[$i].user_id")\"]"

    echo "==> Importing membership for ${USER_NAME} (ID: ${MEMBERSHIP_ID})..."
    (cd "$INFRA_DIR" && terraform import "$IMPORT_ADDR" "$MEMBERSHIP_ID")
  done
fi

# ---------------------------------------------------------------------------
# Run terraform plan to verify
# ---------------------------------------------------------------------------
#echo ""
#echo "==> Running terraform plan..."
#(cd "$INFRA_DIR" && terraform plan)
