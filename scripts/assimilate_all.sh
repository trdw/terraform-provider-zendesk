#!/usr/bin/env bash
# Assimilate every resource of a given classification (a Zendesk admin "list"
# URL) into Terraform. For each resource the Zendesk API returns, calls
# `assimilate.sh <infra-dir> <admin-url> --related-resources=y`. Resources
# already tracked in Terraform state are skipped.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

if [[ $# -lt 2 ]]; then
  echo "Usage: $0 <infra-dir> <zendesk-admin-list-url>"
  echo ""
  echo "Example:"
  echo "  $0 /path/to/infrastructure/sandbox https://my-organization.zendesk.com/admin/objects-rules/tickets/ticket-fields"
  exit 1
fi

INFRA_DIR="$1"
LIST_URL="$2"

if [[ ! -d "$INFRA_DIR" ]]; then
  echo "Error: infrastructure directory '${INFRA_DIR}' does not exist"
  exit 1
fi

PROVIDER_TF="${INFRA_DIR}/provider.tf"
if [[ ! -f "$PROVIDER_TF" ]]; then
  echo "Error: provider.tf not found at ${PROVIDER_TF}"
  exit 1
fi

ZENDESK_SUBDOMAIN=$(grep -oE 'subdomain\s*=\s*"[^"]+"' "$PROVIDER_TF" | grep -oE '"[^"]+"' | tr -d '"')
ZENDESK_EMAIL=$(grep -oE 'email\s*=\s*"[^"]+"' "$PROVIDER_TF" | grep -oE '"[^"]+"' | tr -d '"')

if [[ -z "$ZENDESK_SUBDOMAIN" || -z "$ZENDESK_EMAIL" ]]; then
  echo "Error: could not extract subdomain/email from ${PROVIDER_TF}"
  exit 1
fi

: "${TF_VAR_zendesk_api_token:?Set TF_VAR_zendesk_api_token}"
AUTH="${ZENDESK_EMAIL}/token:${TF_VAR_zendesk_api_token}"
BASE_URL="https://${ZENDESK_SUBDOMAIN}.zendesk.com"

# Sanity check: the admin URL should target the same Zendesk subdomain as
# provider.tf — otherwise we'd list from one tenant and import into another.
URL_HOST=$(echo "$LIST_URL" | sed -E 's|^https?://([^/]+).*|\1|')
if [[ "$URL_HOST" != "${ZENDESK_SUBDOMAIN}.zendesk.com" ]]; then
  echo "Error: URL host (${URL_HOST}) does not match provider.tf subdomain (${ZENDESK_SUBDOMAIN}.zendesk.com)"
  exit 1
fi

# Refuse a per-resource URL (ends with /<id> or /<id>/). Use assimilate.sh
# directly for those.
if echo "$LIST_URL" | grep -qE '/[0-9]+/?$'; then
  echo "Error: this URL points at a specific resource (ends with a numeric ID)."
  echo "Use assimilate.sh for individual resources. assimilate_all.sh expects a"
  echo "classification URL such as:"
  echo "  https://${ZENDESK_SUBDOMAIN}.zendesk.com/admin/objects-rules/tickets/ticket-fields"
  exit 1
fi

# Map the classification URL to a resource type, list endpoint, JSON array
# key, and a per-ID admin URL template. Mirrors assimilate.sh's URL detection;
# order matters (more specific patterns first).
ADMIN_PATH_TPL=""
if echo "$LIST_URL" | grep -qE '/trigger[-_]categor'; then
  RESOURCE_TYPE="trigger_category"
  API_LIST_PATH="/api/v2/trigger_categories.json"
  JQ_ARRAY=".trigger_categories"
  ADMIN_PATH_TPL="/admin/objects-rules/rules/trigger-categories/__ID__"
elif echo "$LIST_URL" | grep -qE '/triggers'; then
  RESOURCE_TYPE="trigger"
  API_LIST_PATH="/api/v2/triggers.json"
  JQ_ARRAY=".triggers"
  ADMIN_PATH_TPL="/admin/objects-rules/rules/triggers/__ID__"
elif echo "$LIST_URL" | grep -qE '/automations'; then
  RESOURCE_TYPE="automation"
  API_LIST_PATH="/api/v2/automations.json"
  JQ_ARRAY=".automations"
  ADMIN_PATH_TPL="/admin/objects-rules/rules/automations/__ID__"
elif echo "$LIST_URL" | grep -qE '/views'; then
  RESOURCE_TYPE="view"
  API_LIST_PATH="/api/v2/views.json"
  JQ_ARRAY=".views"
  ADMIN_PATH_TPL="/admin/workspaces/agent-workspace/views/__ID__"
elif echo "$LIST_URL" | grep -qE '/macros'; then
  RESOURCE_TYPE="macro"
  API_LIST_PATH="/api/v2/macros.json"
  JQ_ARRAY=".macros"
  ADMIN_PATH_TPL="/admin/objects-rules/rules/macros/__ID__"
elif echo "$LIST_URL" | grep -qE '/ticket[-_]forms'; then
  RESOURCE_TYPE="ticket_form"
  API_LIST_PATH="/api/v2/ticket_forms.json"
  JQ_ARRAY=".ticket_forms"
  ADMIN_PATH_TPL="/admin/objects-rules/tickets/ticket-forms/edit/__ID__"
elif echo "$LIST_URL" | grep -qE '/ticket[-_]fields'; then
  RESOURCE_TYPE="ticket_field"
  API_LIST_PATH="/api/v2/ticket_fields.json"
  JQ_ARRAY=".ticket_fields"
  ADMIN_PATH_TPL="/admin/objects-rules/tickets/ticket-fields/__ID__"
elif echo "$LIST_URL" | grep -qE '/user[-_]fields'; then
  RESOURCE_TYPE="user_field"
  API_LIST_PATH="/api/v2/user_fields.json"
  JQ_ARRAY=".user_fields"
  ADMIN_PATH_TPL="/admin/people/configuration/user-fields/__ID__"
elif echo "$LIST_URL" | grep -qE '/groups'; then
  RESOURCE_TYPE="group"
  API_LIST_PATH="/api/v2/groups.json"
  JQ_ARRAY=".groups"
  ADMIN_PATH_TPL="/admin/people/team/groups/__ID__"
elif echo "$LIST_URL" | grep -qE '/roles'; then
  RESOURCE_TYPE="custom_role"
  API_LIST_PATH="/api/v2/custom_roles.json"
  JQ_ARRAY=".custom_roles"
  ADMIN_PATH_TPL="/admin/people/team/roles/__ID__"
elif echo "$LIST_URL" | grep -qE '/members'; then
  # /api/v2/users.json returns end-users too; restrict to agents and admins so
  # we don't try to import every customer ever.
  RESOURCE_TYPE="user"
  API_LIST_PATH="/api/v2/users.json?role[]=agent&role[]=admin"
  JQ_ARRAY=".users"
  ADMIN_PATH_TPL="/admin/people/team/members/__ID__"
elif echo "$LIST_URL" | grep -qE '/brands'; then
  RESOURCE_TYPE="brand"
  API_LIST_PATH="/api/v2/brands.json"
  JQ_ARRAY=".brands"
  ADMIN_PATH_TPL="/admin/account/brand_management/brands/__ID__"
else
  echo "Error: could not determine resource type from URL"
  echo "Supported patterns: triggers, trigger-categories, automations, views,"
  echo "  macros, ticket-forms, ticket-fields, user-fields, groups, roles,"
  echo "  members, brands"
  exit 1
fi

echo "==> Resource type: ${RESOURCE_TYPE}"
echo "==> List endpoint: ${BASE_URL}${API_LIST_PATH}"

# IDs already tracked in terraform state — skip these so we don't double-import.
TF_TYPE="zendesk_${RESOURCE_TYPE}"
echo "==> Reading terraform state..."
EXISTING_IDS=""
TF_STATE_JSON=""
if TF_STATE_JSON=$(cd "$INFRA_DIR" && terraform show -json 2>&1); then
  EXISTING_IDS=$(echo "$TF_STATE_JSON" | jq -r --arg type "$TF_TYPE" '
    .values.root_module.resources[]?
    | select(.type == $type and .mode == "managed")
    | .values.id // empty
  ' 2>/dev/null | sort -u || true)
else
  echo "WARN: 'terraform show -json' failed in ${INFRA_DIR}. Continuing without"
  echo "      state-based skipping (every ID will be passed to assimilate.sh)."
  echo "      Terraform output:"
  echo "$TF_STATE_JSON" | sed 's/^/        /' | head -10
  echo "      You may need to run 'terraform init' in ${INFRA_DIR} first."
fi
EXISTING_COUNT=0
[[ -n "$EXISTING_IDS" ]] && EXISTING_COUNT=$(printf '%s\n' "$EXISTING_IDS" | wc -l | tr -d ' ')
echo "==> ${EXISTING_COUNT} ${RESOURCE_TYPE} already in state"

# Page through the list endpoint and collect every ID.
echo "==> Listing ${RESOURCE_TYPE} from Zendesk..."
SEP="?"
[[ "$API_LIST_PATH" == *"?"* ]] && SEP="&"
next="${BASE_URL}${API_LIST_PATH}${SEP}page[size]=100"
ALL_IDS=""
while [[ -n "$next" && "$next" != "null" ]]; do
  resp=$(curl -fsSg -u "$AUTH" "$next")
  ids=$(echo "$resp" | jq -r "${JQ_ARRAY}[].id")
  ALL_IDS+="${ids}"$'\n'
  next=$(echo "$resp" | jq -r '.links.next // empty')
done
ALL_IDS=$(echo "$ALL_IDS" | sed '/^$/d')
TOTAL=$(echo -n "$ALL_IDS" | grep -c . || true)
echo "==> Found ${TOTAL} ${RESOURCE_TYPE} in Zendesk"

if [[ "$TOTAL" -eq 0 ]]; then
  echo "Nothing to do."
  exit 0
fi

# Iterate.
COUNT=0
SKIPPED=0
FAILED=0
while read -r id; do
  [[ -z "$id" ]] && continue
  COUNT=$((COUNT + 1))

  if echo "$EXISTING_IDS" | grep -qx "$id"; then
    echo
    echo "[${COUNT}/${TOTAL}] Skip ${RESOURCE_TYPE} ${id} (already in state)"
    SKIPPED=$((SKIPPED + 1))
    continue
  fi

  ADMIN_URL="${BASE_URL}${ADMIN_PATH_TPL//__ID__/$id}"
  echo
  echo "===== [${COUNT}/${TOTAL}] ${RESOURCE_TYPE} ${id} ====="
  echo "URL: ${ADMIN_URL}"
  if ! "${SCRIPT_DIR}/assimilate.sh" "${INFRA_DIR}" "${ADMIN_URL}" --related-resources=y; then
    echo "WARN: assimilate.sh failed for ${RESOURCE_TYPE} ${id}; continuing"
    FAILED=$((FAILED + 1))
  fi
done <<< "$ALL_IDS"

echo
echo "==> Done. processed=${COUNT}, skipped=${SKIPPED}, failed=${FAILED}"
