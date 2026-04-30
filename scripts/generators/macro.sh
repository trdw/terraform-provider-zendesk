#!/usr/bin/env bash
# Generator: zendesk_macro
# Maps Zendesk API /api/v2/macros/:id response to Terraform resource

# Format a macro action value for Terraform.
# Strings become quoted strings, arrays become jsonencode([...]).
_format_macro_action_value() {
  local json="$1"
  local index="$2"
  local val_type
  val_type=$(echo "$json" | jq -r ".macro.actions[$index].value | type")

  if [[ "$val_type" == "array" ]]; then
    # See trigger.sh _format_action_value — joining inside jq prevents the
    # external sed from corrupting strings that contain commas, and the
    # backslash-doubling lets @json escapes survive `echo -e`.
    local joined
    joined=$(echo "$json" | jq -r ".macro.actions[$index].value | map(@json) | join(\", \")")
    joined="${joined//\\/\\\\}"
    echo "jsonencode([${joined}])"
  else
    local val
    val=$(echo "$json" | jq -r ".macro.actions[$index].value // \"\"")
    echo "\"$(hcl_escape "$val")\""
  fi
}

generate_macro() {
  local json="$1"
  local title active description res_name

  title=$(echo "$json" | jq -r '.macro.title')
  active=$(echo "$json" | jq -r '.macro.active')
  description=$(echo "$json" | jq -r '.macro.description // empty')
  res_name=$(to_snake_case "$title")

  local tf=""
  tf+="resource \"zendesk_macro\" \"${res_name}\" {\n"
  tf+="  title  = \"$(hcl_escape "$title")\"\n"
  tf+="  active = ${active}\n"
  if [[ -n "$description" ]]; then
    tf+="  description = \"$(hcl_escape "$description")\"\n"
  fi

  # Actions
  local action_count
  action_count=$(echo "$json" | jq '.macro.actions | length')
  if [[ "$action_count" -gt 0 ]]; then
    tf+="\n  actions = [\n"
    for i in $(seq 0 $((action_count - 1))); do
      local field formatted_value
      field=$(echo "$json" | jq -r ".macro.actions[$i].field")
      formatted_value=$(_format_macro_action_value "$json" "$i")
      tf+="    {\n"
      tf+="      field = \"$(hcl_escape "$field")\"\n"
      tf+="      value = ${formatted_value}\n"
      tf+="    },\n"
    done
    tf+="  ]\n"
  fi

  # Restriction
  local has_restriction
  has_restriction=$(echo "$json" | jq '.macro.restriction // null | type == "object" and has("type")')
  if [[ "$has_restriction" == "true" ]]; then
    local r_type r_id
    r_type=$(echo "$json" | jq -r '.macro.restriction.type')
    r_id=$(echo "$json" | jq -r '.macro.restriction.id')
    tf+="\n  restriction = {\n"
    tf+="    type = \"$(hcl_escape "$r_type")\"\n"
    tf+="    id   = ${r_id}\n"
    tf+="  }\n"
  fi

  tf+="}"

  echo -e "$tf"
}
