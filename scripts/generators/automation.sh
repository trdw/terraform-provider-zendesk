#!/usr/bin/env bash
# Generator: zendesk_automation
# Maps Zendesk API /api/v2/automations/:id response to Terraform resource

# Map of automation condition/action field names to the Zendesk resource
# type they reference. Returns empty for fields that aren't ID references.
_automation_field_to_resource_type() {
  case "$1" in
    group_id)                               echo "group" ;;
    brand_id)                               echo "brand" ;;
    ticket_form_id)                         echo "ticket_form" ;;
    assignee_id|requester_id|current_user_id) echo "user" ;;
    *)                                      echo "" ;;
  esac
}

_format_automation_referenced_value() {
  local field="$1" value="$2" maps="$3"
  local rtype
  rtype=$(_automation_field_to_resource_type "$field")
  if [[ -n "$rtype" && -n "$value" && "$value" != "null" ]]; then
    local addr
    addr=$(echo "$maps" | jq -r --arg t "$rtype" --arg id "$value" '.[$t][$id] // empty')
    if [[ -n "$addr" ]]; then
      printf '%s.id' "$addr"
      return
    fi
  fi
  printf '"%s"' "$(hcl_escape "$value")"
}

_format_automation_action_value() {
  local json="$1"
  local index="$2"
  local maps="${3:-}"
  [[ -z "$maps" ]] && maps='{}'
  local val_type
  val_type=$(echo "$json" | jq -r ".automation.actions[$index].value | type")

  if [[ "$val_type" == "array" ]]; then
    # See trigger.sh _format_action_value — joining inside jq prevents the
    # external sed from corrupting strings that contain commas, and the
    # backslash-doubling lets @json escapes survive `echo -e`.
    local joined
    joined=$(echo "$json" | jq -r ".automation.actions[$index].value | map(@json) | join(\", \")")
    joined="${joined//\\/\\\\}"
    echo "jsonencode([${joined}])"
  else
    local field val
    field=$(echo "$json" | jq -r ".automation.actions[$index].field")
    val=$(echo "$json" | jq -r ".automation.actions[$index].value // \"\"")
    _format_automation_referenced_value "$field" "$val" "$maps"
  fi
}

generate_automation() {
  # $2 (optional): JSON object mapping resource type and id to an HCL address.
  # See trigger.sh `generate_trigger` for shape.
  local json="$1"
  local maps="${2:-}"
  [[ -z "$maps" ]] && maps='{}'
  local title active position res_name

  title=$(echo "$json" | jq -r '.automation.title')
  active=$(echo "$json" | jq -r '.automation.active')
  position=$(echo "$json" | jq -r '.automation.position // empty')
  res_name=$(to_snake_case "$title")

  local tf=""
  tf+="resource \"zendesk_automation\" \"${res_name}\" {\n"
  tf+="  title  = \"$(hcl_escape "$title")\"\n"
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
      formatted_value=$(_format_automation_action_value "$json" "$i" "$maps")
      tf+="    {\n"
      tf+="      field = \"$(hcl_escape "$field")\"\n"
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
      tf+="      field    = \"$(hcl_escape "$field")\"\n"
      tf+="      operator = \"$(hcl_escape "$operator")\"\n"
      tf+="      value    = $(_format_automation_referenced_value "$field" "$value" "$maps")\n"
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
      tf+="      field    = \"$(hcl_escape "$field")\"\n"
      tf+="      operator = \"$(hcl_escape "$operator")\"\n"
      tf+="      value    = $(_format_automation_referenced_value "$field" "$value" "$maps")\n"
      tf+="    },\n"
    done
    tf+="  ]\n"
  fi

  tf+="}"

  echo -e "$tf"
}
