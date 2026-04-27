#!/usr/bin/env bash
# Generator: zendesk_brand
# Maps Zendesk API /api/v2/brands/:id response to Terraform resource

generate_brand() {
  local json="$1"
  local name subdomain active brand_url host_mapping has_help_center signature_template res_name

  name=$(echo "$json" | jq -r '.brand.name')
  subdomain=$(echo "$json" | jq -r '.brand.subdomain')
  active=$(jq_bool "$json" '.brand.active' true)
  brand_url=$(echo "$json" | jq -r '.brand.brand_url // empty')
  host_mapping=$(echo "$json" | jq -r '.brand.host_mapping // empty')
  has_help_center=$(echo "$json" | jq -r '.brand.has_help_center // empty')
  signature_template=$(echo "$json" | jq -r '.brand.signature_template // empty')
  res_name=$(to_snake_case "$name")

  local tf=""
  tf+="resource \"zendesk_brand\" \"${res_name}\" {\n"
  tf+="  name      = \"$(hcl_escape "$name")\"\n"
  tf+="  subdomain = \"$(hcl_escape "$subdomain")\"\n"
  tf+="  active    = ${active}\n"
  if [[ -n "$brand_url" ]]; then
    tf+="  brand_url = \"$(hcl_escape "$brand_url")\"\n"
  fi
  if [[ -n "$host_mapping" ]]; then
    tf+="  host_mapping = \"$(hcl_escape "$host_mapping")\"\n"
  fi
  if [[ -n "$has_help_center" && "$has_help_center" != "null" ]]; then
    tf+="  has_help_center = ${has_help_center}\n"
  fi
  if [[ -n "$signature_template" ]]; then
    tf+="  signature_template = \"$(hcl_escape "$signature_template")\"\n"
  fi
  tf+="}"

  echo -e "$tf"
}
