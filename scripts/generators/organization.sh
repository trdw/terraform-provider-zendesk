#!/usr/bin/env bash
# Generator: zendesk_organization
# Maps Zendesk API /api/v2/organizations/:id response to Terraform resource

generate_organization() {
  local json="$1"
  local name res_name
  name=$(echo "$json" | jq -r '.organization.name')
  res_name=$(to_snake_case "$name")

  local tf=""
  tf+="resource \"zendesk_organization\" \"${res_name}\" {\n"
  tf+="  name = \"$(hcl_escape "$name")\"\n"

  # Optional scalar strings — emit only when present.
  local details notes external_id
  details=$(echo "$json" | jq -r '.organization.details // empty')
  notes=$(echo "$json" | jq -r '.organization.notes // empty')
  external_id=$(echo "$json" | jq -r '.organization.external_id // empty')
  [[ -n "$details" ]] && tf+="  details     = \"$(hcl_escape "$details")\"\n"
  [[ -n "$notes" ]] && tf+="  notes       = \"$(hcl_escape "$notes")\"\n"
  [[ -n "$external_id" ]] && tf+="  external_id = \"$(hcl_escape "$external_id")\"\n"

  # group_id (nullable integer).
  local group_id
  group_id=$(echo "$json" | jq -r '.organization.group_id // empty')
  [[ -n "$group_id" ]] && tf+="  group_id    = ${group_id}\n"

  # Booleans (default false when absent).
  local shared_comments shared_tickets
  shared_comments=$(jq_bool "$json" '.organization.shared_comments' false)
  shared_tickets=$(jq_bool "$json" '.organization.shared_tickets' false)
  tf+="  shared_comments = ${shared_comments}\n"
  tf+="  shared_tickets  = ${shared_tickets}\n"

  # domain_names / tags (arrays of strings). See macro.sh for the @json + sed
  # interplay; the backslash-doubling lets @json escapes survive `echo -e`.
  local dn_count tag_count joined
  dn_count=$(echo "$json" | jq '(.organization.domain_names // []) | length')
  if [[ "$dn_count" -gt 0 ]]; then
    joined=$(echo "$json" | jq -r '.organization.domain_names | map(@json) | join(", ")')
    joined="${joined//\\/\\\\}"
    tf+="  domain_names = [${joined}]\n"
  fi
  tag_count=$(echo "$json" | jq '(.organization.tags // []) | length')
  if [[ "$tag_count" -gt 0 ]]; then
    joined=$(echo "$json" | jq -r '.organization.tags | map(@json) | join(", ")')
    joined="${joined//\\/\\\\}"
    tf+="  tags = [${joined}]\n"
  fi

  # organization_fields (object key -> scalar). The provider models the values
  # as strings, so every non-null value is emitted quoted.
  local of_lines
  of_lines=$(echo "$json" | jq -r '
    (.organization.organization_fields // {})
    | to_entries
    | map(select(.value != null))
    | map("    \(.key|@json) = \(.value|tostring|@json)")
    | join("\n")')
  if [[ -n "$of_lines" ]]; then
    of_lines="${of_lines//\\/\\\\}"
    tf+="\n  organization_fields = {\n"
    tf+="${of_lines}\n"
    tf+="  }\n"
  fi

  tf+="}"
  echo -e "$tf"
}
