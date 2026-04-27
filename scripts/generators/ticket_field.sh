#!/usr/bin/env bash
# Generator: zendesk_ticket_field
# Maps Zendesk API /api/v2/ticket_fields/:id response to Terraform resource

generate_ticket_field() {
  local json="$1"
  local type title res_name

  type=$(echo "$json" | jq -r '.ticket_field.type')
  title=$(echo "$json" | jq -r '.ticket_field.title')
  res_name=$(to_snake_case "$title")

  local tf=""
  tf+="resource \"zendesk_ticket_field\" \"${res_name}\" {\n"
  tf+="  type  = \"$(hcl_escape "$type")\"\n"
  tf+="  title = \"$(hcl_escape "$title")\"\n"

  _emit_field_str() {
    local jq_path="$1" tf_name="$2"
    local val
    val=$(echo "$json" | jq -r "${jq_path} // empty")
    if [[ -n "$val" && "$val" != "null" ]]; then
      tf+="  ${tf_name} = \"$(hcl_escape "$val")\"\n"
    fi
  }

  _emit_field_bool() {
    local jq_path="$1" tf_name="$2"
    local val
    val=$(echo "$json" | jq -r "${jq_path} // empty")
    if [[ -n "$val" && "$val" != "null" ]]; then
      tf+="  ${tf_name} = ${val}\n"
    fi
  }

  _emit_field_int() {
    local jq_path="$1" tf_name="$2"
    local val
    val=$(echo "$json" | jq -r "${jq_path} // empty")
    if [[ -n "$val" && "$val" != "null" ]]; then
      tf+="  ${tf_name} = ${val}\n"
    fi
  }

  _emit_field_str  ".ticket_field.title_in_portal"          "title_in_portal"
  _emit_field_str  ".ticket_field.description"              "description"
  _emit_field_str  ".ticket_field.agent_description"        "agent_description"
  _emit_field_bool ".ticket_field.active"                   "active"
  _emit_field_bool ".ticket_field.required"                 "required"
  _emit_field_bool ".ticket_field.required_in_portal"       "required_in_portal"
  _emit_field_bool ".ticket_field.visible_in_portal"        "visible_in_portal"
  _emit_field_bool ".ticket_field.editable_in_portal"       "editable_in_portal"
  _emit_field_bool ".ticket_field.collapsed_for_agents"     "collapsed_for_agents"
  _emit_field_int  ".ticket_field.position"                 "position"
  # regexp_for_validation only applies when type is "regexp".
  if [[ "$type" == "regexp" ]]; then
    local regexp_val
    regexp_val=$(echo "$json" | jq -r '.ticket_field.regexp_for_validation // empty')
    if [[ -n "$regexp_val" && "$regexp_val" != "null" ]]; then
      tf+="  regexp_for_validation = \"$(hcl_escape "$regexp_val")\"\n"
    fi
  fi
  _emit_field_str  ".ticket_field.tag"                      "tag"
  _emit_field_int  ".ticket_field.sub_type_id"              "sub_type_id"
  _emit_field_str  ".ticket_field.relationship_target_type" "relationship_target_type"

  # custom_field_options
  local opt_count
  opt_count=$(echo "$json" | jq '.ticket_field.custom_field_options | length // 0')
  if [[ "$opt_count" -gt 0 ]]; then
    tf+="\n  custom_field_options = [\n"
    for i in $(seq 0 $((opt_count - 1))); do
      local opt_name opt_value
      opt_name=$(echo "$json"  | jq -r ".ticket_field.custom_field_options[$i].name")
      opt_value=$(echo "$json" | jq -r ".ticket_field.custom_field_options[$i].value")
      tf+="    {\n"
      tf+="      name  = \"$(hcl_escape "$opt_name")\"\n"
      tf+="      value = \"$(hcl_escape "$opt_value")\"\n"
      tf+="    },\n"
    done
    tf+="  ]\n"
  fi

  tf+="}"

  echo -e "$tf"
}
