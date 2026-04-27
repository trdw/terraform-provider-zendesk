#!/usr/bin/env bash
# Compare the ticket_fields in a Zendesk subdomain against what is tracked in
# the corresponding terraform state. Reports membership diff and points at
# `terraform plan` for content drift on fields present in both places.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

if [[ $# -lt 1 ]]; then
  echo "Usage: $0 <infra-dir>"
  echo ""
  echo "Example:"
  echo "  $0 /path/to/infrastructure/sandbox"
  exit 1
fi

INFRA_DIR="$1"
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

echo "==> Comparing ticket_fields for subdomain: ${ZENDESK_SUBDOMAIN}"

ZD=$(mktemp)
TF=$(mktemp)
trap 'rm -f "$ZD" "$TF"' EXIT

# 1. Pull every ticket_field from Zendesk (cursor pagination).
next="${BASE_URL}/api/v2/ticket_fields.json?page[size]=100"
while [[ -n "$next" && "$next" != "null" ]]; do
  resp=$(curl -fsSg -u "$AUTH" "$next")
  echo "$resp" | jq -c '.ticket_fields[]' >> "$ZD"
  next=$(echo "$resp" | jq -r '.links.next // empty')
done
echo "Zendesk: $(wc -l < "$ZD" | tr -d ' ') ticket_fields"

# 2. Collect ticket_field IDs from local terraform state (resources + data
# sources). One `terraform show -json` is dramatically faster than calling
# `terraform state show` per address against an S3 backend.
echo "Reading terraform state..."
(cd "$INFRA_DIR" && terraform show -json 2>/dev/null) | jq -r '
  .values.root_module.resources[]?
  | select(.type == "zendesk_ticket_field")
  | "\(.values.id)\t\(.address)"
' > "$TF"
echo "Terraform state: $(wc -l < "$TF" | tr -d ' ') tracked"

# 3. Membership diff.
echo
echo "===== In Zendesk but NOT in terraform state ====="
comm -23 <(jq -r '.id' "$ZD" | sort -un) <(cut -f1 "$TF" | sort -un) | while read -r id; do
  jq -r --arg id "$id" 'select((.id|tostring)==$id) | "  \(.id)\t\(.title)\t(type=\(.type), removable=\(.removable))"' "$ZD"
done

echo
echo "===== In terraform state but NOT in Zendesk (deleted upstream?) ====="
comm -13 <(jq -r '.id' "$ZD" | sort -un) <(cut -f1 "$TF" | sort -un) | while read -r id; do
  grep "^${id}	" "$TF" | sed 's/^/  /'
done

echo
echo "===== Content drift on fields tracked in both ====="
echo "Run from ${INFRA_DIR}:"
echo "  terraform plan | grep -A2 'zendesk_ticket_field\\.'"
