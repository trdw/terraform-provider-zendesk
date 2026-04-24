#!/usr/bin/env bash
# Generator: zendesk_user_field
# Maps Zendesk API /api/v2/user_fields/:id response to Terraform resource

generate_user_field() {
  local json="$1"
  local key type title res_name

  key=$(echo "$json"   | jq -r '.user_field.key')
  type=$(echo "$json"  | jq -r '.user_field.type')
  title=$(echo "$json" | jq -r '.user_field.title')
  res_name=$(to_snake_case "$key")

  local tf=""
  tf+="resource \"zendesk_user_field\" \"${res_name}\" {\n"
  tf+="  key   = \"${key}\"\n"
  tf+="  type  = \"${type}\"\n"
  tf+="  title = \"${title}\"\n"

  _emit_ufield_str() {
    local jq_path="$1" tf_name="$2"
    local val
    val=$(echo "$json" | jq -r "${jq_path} // empty")
    if [[ -n "$val" && "$val" != "null" ]]; then
      tf+="  ${tf_name} = \"${val}\"\n"
    fi
  }

  _emit_ufield_bool() {
    local jq_path="$1" tf_name="$2"
    local val
    val=$(echo "$json" | jq -r "${jq_path} // empty")
    if [[ -n "$val" && "$val" != "null" ]]; then
      tf+="  ${tf_name} = ${val}\n"
    fi
  }

  _emit_ufield_int() {
    local jq_path="$1" tf_name="$2"
    local val
    val=$(echo "$json" | jq -r "${jq_path} // empty")
    if [[ -n "$val" && "$val" != "null" ]]; then
      tf+="  ${tf_name} = ${val}\n"
    fi
  }

  _emit_ufield_str  ".user_field.description"              "description"
  _emit_ufield_bool ".user_field.active"                   "active"
  _emit_ufield_int  ".user_field.position"                 "position"
  _emit_ufield_str  ".user_field.regexp_for_validation"    "regexp_for_validation"
  _emit_ufield_str  ".user_field.tag"                      "tag"
  _emit_ufield_str  ".user_field.relationship_target_type" "relationship_target_type"

  # custom_field_options
  local opt_count
  opt_count=$(echo "$json" | jq '.user_field.custom_field_options | length // 0')
  if [[ "$opt_count" -gt 0 ]]; then
    tf+="\n  custom_field_options = [\n"
    for i in $(seq 0 $((opt_count - 1))); do
      local opt_name opt_value
      opt_name=$(echo "$json"  | jq -r ".user_field.custom_field_options[$i].name")
      opt_value=$(echo "$json" | jq -r ".user_field.custom_field_options[$i].value")
      tf+="    {\n"
      tf+="      name  = \"${opt_name}\"\n"
      tf+="      value = \"${opt_value}\"\n"
      tf+="    },\n"
    done
    tf+="  ]\n"
  fi

  tf+="}"

  echo -e "$tf"
}
