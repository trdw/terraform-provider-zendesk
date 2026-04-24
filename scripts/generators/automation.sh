#!/usr/bin/env bash
# Generator: zendesk_automation
# Maps Zendesk API /api/v2/automations/:id response to Terraform resource

_format_automation_action_value() {
  local json="$1"
  local index="$2"
  local val_type
  val_type=$(echo "$json" | jq -r ".automation.actions[$index].value | type")

  if [[ "$val_type" == "array" ]]; then
    local elements
    elements=$(echo "$json" | jq -r ".automation.actions[$index].value[] | @json")
    local joined
    joined=$(echo "$elements" | paste -sd',' - | sed 's/,/, /g')
    echo "jsonencode([${joined}])"
  else
    local val
    val=$(echo "$json" | jq -r ".automation.actions[$index].value // \"\"")
    echo "\"${val}\""
  fi
}

generate_automation() {
  local json="$1"
  local title active position res_name

  title=$(echo "$json" | jq -r '.automation.title')
  active=$(echo "$json" | jq -r '.automation.active')
  position=$(echo "$json" | jq -r '.automation.position // empty')
  res_name=$(to_snake_case "$title")

  local tf=""
  tf+="resource \"zendesk_automation\" \"${res_name}\" {\n"
  tf+="  title  = \"${title}\"\n"
  tf+="  active = ${active}\n"
  if [[ -n "$position" ]]; then
    tf+="  position = ${position}\n"
  fi

  # Actions
  local action_count
  action_count=$(echo "$json" | jq '.automation.actions | length')
  if [[ "$action_count" -gt 0 ]]; then
    tf+="\n  actions = [\n"
    for i in $(seq 0 $((action_count - 1))); do
      local field formatted_value
      field=$(echo "$json" | jq -r ".automation.actions[$i].field")
      formatted_value=$(_format_automation_action_value "$json" "$i")
      tf+="    {\n"
      tf+="      field = \"${field}\"\n"
      tf+="      value = ${formatted_value}\n"
      tf+="    },\n"
    done
    tf+="  ]\n"
  fi

  # Conditions: all
  local all_count
  all_count=$(echo "$json" | jq '.automation.conditions.all | length')
  if [[ "$all_count" -gt 0 ]]; then
    tf+="\n  condition_all = [\n"
    for i in $(seq 0 $((all_count - 1))); do
      local field operator value
      field=$(echo "$json" | jq -r ".automation.conditions.all[$i].field")
      operator=$(echo "$json" | jq -r ".automation.conditions.all[$i].operator")
      value=$(echo "$json" | jq -r ".automation.conditions.all[$i].value // \"\"")
      tf+="    {\n"
      tf+="      field    = \"${field}\"\n"
      tf+="      operator = \"${operator}\"\n"
      tf+="      value    = \"${value}\"\n"
      tf+="    },\n"
    done
    tf+="  ]\n"
  fi

  # Conditions: any
  local any_count
  any_count=$(echo "$json" | jq '.automation.conditions.any | length')
  if [[ "$any_count" -gt 0 ]]; then
    tf+="\n  condition_any = [\n"
    for i in $(seq 0 $((any_count - 1))); do
      local field operator value
      field=$(echo "$json" | jq -r ".automation.conditions.any[$i].field")
      operator=$(echo "$json" | jq -r ".automation.conditions.any[$i].operator")
      value=$(echo "$json" | jq -r ".automation.conditions.any[$i].value // \"\"")
      tf+="    {\n"
      tf+="      field    = \"${field}\"\n"
      tf+="      operator = \"${operator}\"\n"
      tf+="      value    = \"${value}\"\n"
      tf+="    },\n"
    done
    tf+="  ]\n"
  fi

  tf+="}"

  echo -e "$tf"
}
