#!/usr/bin/env bash
# Generator: zendesk_trigger
# Maps Zendesk API /api/v2/triggers/:id response to Terraform resource

# Format a trigger action value for Terraform.
# Strings become quoted strings, arrays become jsonencode([...]).
_format_action_value() {
  local json="$1"
  local index="$2"
  local val_type
  val_type=$(echo "$json" | jq -r ".trigger.actions[$index].value | type")

  if [[ "$val_type" == "array" ]]; then
    # Build jsonencode([...]) with each element as a quoted string
    local elements
    elements=$(echo "$json" | jq -r ".trigger.actions[$index].value[] | @json")
    local joined
    joined=$(echo "$elements" | paste -sd',' - | sed 's/,/, /g')
    echo "jsonencode([${joined}])"
  else
    local val
    val=$(echo "$json" | jq -r ".trigger.actions[$index].value // \"\"")
    echo "\"${val}\""
  fi
}

generate_trigger() {
  local json="$1"
  local title active description category_id res_name

  title=$(echo "$json" | jq -r '.trigger.title')
  active=$(echo "$json" | jq -r '.trigger.active')
  description=$(echo "$json" | jq -r '.trigger.description // ""')
  category_id=$(echo "$json" | jq -r '.trigger.category_id // empty')
  res_name=$(to_snake_case "$title")

  local tf=""
  tf+="resource \"zendesk_trigger\" \"${res_name}\" {\n"
  tf+="  title       = \"${title}\"\n"
  tf+="  active      = ${active}\n"
  if [[ -n "$description" ]]; then
    tf+="  description = \"${description}\"\n"
  fi
  if [[ -n "$category_id" ]]; then
    tf+="  category_id = \"${category_id}\"\n"
  fi

  # Actions
  local action_count
  action_count=$(echo "$json" | jq '.trigger.actions | length')
  if [[ "$action_count" -gt 0 ]]; then
    tf+="\n  actions = [\n"
    for i in $(seq 0 $((action_count - 1))); do
      local field formatted_value
      field=$(echo "$json" | jq -r ".trigger.actions[$i].field")
      formatted_value=$(_format_action_value "$json" "$i")
      tf+="    {\n"
      tf+="      field = \"${field}\"\n"
      tf+="      value = ${formatted_value}\n"
      tf+="    },\n"
    done
    tf+="  ]\n"
  fi

  # Conditions: all
  local all_count
  all_count=$(echo "$json" | jq '.trigger.conditions.all | length')
  if [[ "$all_count" -gt 0 ]]; then
    tf+="\n  condition_all = [\n"
    for i in $(seq 0 $((all_count - 1))); do
      local field operator value
      field=$(echo "$json" | jq -r ".trigger.conditions.all[$i].field")
      operator=$(echo "$json" | jq -r ".trigger.conditions.all[$i].operator")
      value=$(echo "$json" | jq -r ".trigger.conditions.all[$i].value // \"\"")
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
  any_count=$(echo "$json" | jq '.trigger.conditions.any | length')
  if [[ "$any_count" -gt 0 ]]; then
    tf+="\n  condition_any = [\n"
    for i in $(seq 0 $((any_count - 1))); do
      local field operator value
      field=$(echo "$json" | jq -r ".trigger.conditions.any[$i].field")
      operator=$(echo "$json" | jq -r ".trigger.conditions.any[$i].operator")
      value=$(echo "$json" | jq -r ".trigger.conditions.any[$i].value // \"\"")
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
