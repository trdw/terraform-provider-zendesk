#!/usr/bin/env bash
# Generator: zendesk_trigger_category
# Maps Zendesk API /api/v2/trigger_categories/:id response to Terraform resource

generate_trigger_category() {
  local json="$1"
  local name position res_name

  name=$(echo "$json" | jq -r '.trigger_category.name')
  position=$(echo "$json" | jq -r '.trigger_category.position // empty')
  res_name=$(to_snake_case "$name")

  local tf=""
  tf+="resource \"zendesk_trigger_category\" \"${res_name}\" {\n"
  tf+="  name = \"${name}\"\n"
  if [[ -n "$position" ]]; then
    tf+="  position = ${position}\n"
  fi
  tf+="}"

  echo -e "$tf"
}
