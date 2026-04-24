#!/usr/bin/env bash
# Generator: zendesk_ticket_form
# Maps Zendesk API /api/v2/ticket_forms/:id response to Terraform resource

generate_ticket_form() {
  local json="$1"
  local name display_name active default end_user_visible position in_all_brands res_name

  name=$(echo "$json" | jq -r '.ticket_form.name')
  display_name=$(echo "$json" | jq -r '.ticket_form.display_name // empty')
  active=$(echo "$json" | jq -r '.ticket_form.active // true')
  default=$(echo "$json" | jq -r '.ticket_form.default // false')
  end_user_visible=$(echo "$json" | jq -r '.ticket_form.end_user_visible // true')
  position=$(echo "$json" | jq -r '.ticket_form.position // empty')
  in_all_brands=$(echo "$json" | jq -r '.ticket_form.in_all_brands // false')
  res_name=$(to_snake_case "$name")

  local tf=""
  tf+="resource \"zendesk_ticket_form\" \"${res_name}\" {\n"
  tf+="  name             = \"${name}\"\n"
  if [[ -n "$display_name" ]]; then
    tf+="  display_name     = \"${display_name}\"\n"
  fi
  tf+="  active           = ${active}\n"
  tf+="  default          = ${default}\n"
  tf+="  end_user_visible = ${end_user_visible}\n"
  tf+="  in_all_brands    = ${in_all_brands}\n"
  if [[ -n "$position" ]]; then
    tf+="  position         = ${position}\n"
  fi

  # ticket_field_ids
  local field_ids
  field_ids=$(echo "$json" | jq -r '[.ticket_form.ticket_field_ids[]?] | map(tostring) | join(", ")')
  if [[ -n "$field_ids" ]]; then
    tf+="\n  ticket_field_ids = [${field_ids}]\n"
  fi

  # restricted_brand_ids
  local brand_ids
  brand_ids=$(echo "$json" | jq -r '[.ticket_form.restricted_brand_ids[]?] | map(tostring) | join(", ")')
  if [[ -n "$brand_ids" ]]; then
    tf+="  restricted_brand_ids = [${brand_ids}]\n"
  fi

  tf+="}"

  echo -e "$tf"
}
