#!/usr/bin/env bash
# Generator: zendesk_trigger
# Maps Zendesk API /api/v2/triggers/:id response to Terraform resource

# Map of trigger condition/action field names to the Zendesk resource type
# they reference. Returns empty for fields that aren't ID references.
_field_to_resource_type() {
  case "$1" in
    group_id)                               echo "group" ;;
    brand_id)                               echo "brand" ;;
    ticket_form_id)                         echo "ticket_form" ;;
    assignee_id|requester_id|current_user_id) echo "user" ;;
    *)                                      echo "" ;;
  esac
}

# Format a single (field, value) pair as either a Terraform reference
# (`<addr>.id`, unquoted) when the value matches an entry in resource_maps,
# or as a quoted HCL-escaped scalar.
_format_referenced_value() {
  local field="$1" value="$2" maps="$3"
  local rtype
  rtype=$(_field_to_resource_type "$field")
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

# Format a trigger action value for Terraform.
# Strings become quoted strings (or references for known ID fields), arrays
# become jsonencode([...]).
_format_action_value() {
  local json="$1"
  local index="$2"
  local maps="${3:-}"
  [[ -z "$maps" ]] && maps='{}'
  local val_type
  val_type=$(echo "$json" | jq -r ".trigger.actions[$index].value | type")

  if [[ "$val_type" == "array" ]]; then
    # Build the array contents inside jq so each element stays a discrete
    # JSON string — joining with sed corrupts strings that legitimately
    # contain commas (e.g. "Thank you,<br>" would gain a stray space).
    # Then double every backslash so the 2-char @json escapes (`\n`, `\\`,
    # `\"`, `\t`) survive the final `echo -e` and land in the .tf as
    # HCL-correct escapes.
    local joined
    joined=$(echo "$json" | jq -r ".trigger.actions[$index].value | map(@json) | join(\", \")")
    joined="${joined//\\/\\\\}"
    echo "jsonencode([${joined}])"
  else
    local field val
    field=$(echo "$json" | jq -r ".trigger.actions[$index].field")
    val=$(echo "$json" | jq -r ".trigger.actions[$index].value // \"\"")
    _format_referenced_value "$field" "$val" "$maps"
  fi
}

generate_trigger() {
  # $2 (optional): JSON object mapping resource type and id to an HCL address,
  # used to emit Terraform references in place of raw IDs in condition/action
  # values. Shape:
  #   { "group": {"<id>": "zendesk_group.<name>"},
  #     "brand": {"<id>": "zendesk_brand.<name>"}, ... }
  local json="$1"
  local maps="${2:-}"
  [[ -z "$maps" ]] && maps='{}'
  local title active description category_id res_name

  title=$(echo "$json" | jq -r '.trigger.title')
  active=$(echo "$json" | jq -r '.trigger.active')
  description=$(echo "$json" | jq -r '.trigger.description // ""')
  category_id=$(echo "$json" | jq -r '.trigger.category_id // empty')
  res_name=$(to_snake_case "$title")

  local tf=""
  tf+="resource \"zendesk_trigger\" \"${res_name}\" {\n"
  tf+="  title       = \"$(hcl_escape "$title")\"\n"
  tf+="  active      = ${active}\n"
  if [[ -n "$description" ]]; then
    tf+="  description = \"$(hcl_escape "$description")\"\n"
  fi
  if [[ -n "$category_id" ]]; then
    tf+="  category_id = \"$(hcl_escape "$category_id")\"\n"
  fi

  # Actions
  local action_count
  action_count=$(echo "$json" | jq '.trigger.actions | length')
  if [[ "$action_count" -gt 0 ]]; then
    tf+="\n  actions = [\n"
    for i in $(seq 0 $((action_count - 1))); do
      local field formatted_value
      field=$(echo "$json" | jq -r ".trigger.actions[$i].field")
      formatted_value=$(_format_action_value "$json" "$i" "$maps")
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
      tf+="      field    = \"$(hcl_escape "$field")\"\n"
      tf+="      operator = \"$(hcl_escape "$operator")\"\n"
      tf+="      value    = $(_format_referenced_value "$field" "$value" "$maps")\n"
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
      tf+="      field    = \"$(hcl_escape "$field")\"\n"
      tf+="      operator = \"$(hcl_escape "$operator")\"\n"
      tf+="      value    = $(_format_referenced_value "$field" "$value" "$maps")\n"
      tf+="    },\n"
    done
    tf+="  ]\n"
  fi

  tf+="}"

  echo -e "$tf"
}
