#!/usr/bin/env bash
# Generator: zendesk_view
# Maps Zendesk API /api/v2/views/:id response to Terraform resource

generate_view() {
  local json="$1"
  local title active res_name

  title=$(echo "$json" | jq -r '.view.title')
  active=$(echo "$json" | jq -r '.view.active')
  res_name=$(to_snake_case "$title")

  local tf=""
  tf+="resource \"zendesk_view\" \"${res_name}\" {\n"
  tf+="  title  = \"$(hcl_escape "$title")\"\n"
  tf+="  active = ${active}\n"

  # Conditions: all
  local all_count
  all_count=$(echo "$json" | jq '.view.conditions.all | length')
  if [[ "$all_count" -gt 0 ]]; then
    tf+="\n  all = [\n"
    for i in $(seq 0 $((all_count - 1))); do
      local field operator value
      field=$(echo "$json" | jq -r ".view.conditions.all[$i].field")
      operator=$(echo "$json" | jq -r ".view.conditions.all[$i].operator")
      value=$(echo "$json" | jq -r ".view.conditions.all[$i].value // \"\"")
      tf+="    {\n"
      tf+="      field    = \"$(hcl_escape "$field")\"\n"
      tf+="      operator = \"$(hcl_escape "$operator")\"\n"
      tf+="      value    = \"$(hcl_escape "$value")\"\n"
      tf+="    },\n"
    done
    tf+="  ]\n"
  fi

  # Conditions: any
  local any_count
  any_count=$(echo "$json" | jq '.view.conditions.any | length')
  if [[ "$any_count" -gt 0 ]]; then
    tf+="\n  any = [\n"
    for i in $(seq 0 $((any_count - 1))); do
      local field operator value
      field=$(echo "$json" | jq -r ".view.conditions.any[$i].field")
      operator=$(echo "$json" | jq -r ".view.conditions.any[$i].operator")
      value=$(echo "$json" | jq -r ".view.conditions.any[$i].value // \"\"")
      tf+="    {\n"
      tf+="      field    = \"$(hcl_escape "$field")\"\n"
      tf+="      operator = \"$(hcl_escape "$operator")\"\n"
      tf+="      value    = \"$(hcl_escape "$value")\"\n"
      tf+="    },\n"
    done
    tf+="  ]\n"
  fi

  # Execution: columns (API returns objects with "id" field)
  local cols
  cols=$(echo "$json" | jq -r '[.view.execution.columns[]? | .id // .] | map(tostring) | map("\"" + . + "\"") | join(", ")')
  if [[ -n "$cols" ]]; then
    tf+="\n  columns    = [${cols}]\n"
  fi

  # Execution: sort
  local sort_by sort_order
  sort_by=$(echo "$json" | jq -r '.view.execution.sort_by // empty')
  sort_order=$(echo "$json" | jq -r '.view.execution.sort_order // empty')
  if [[ -n "$sort_by" ]]; then
    tf+="  sort_by    = \"${sort_by}\"\n"
  fi
  if [[ -n "$sort_order" ]]; then
    tf+="  sort_order = \"${sort_order}\"\n"
  fi

  # Execution: group
  local group_by group_order
  group_by=$(echo "$json" | jq -r '.view.execution.group_by // empty')
  group_order=$(echo "$json" | jq -r '.view.execution.group_order // empty')
  if [[ -n "$group_by" ]]; then
    tf+="\n  group_by    = \"${group_by}\"\n"
  fi
  if [[ -n "$group_order" ]]; then
    tf+="  group_order = \"${group_order}\"\n"
  fi

  # Restriction
  local has_restriction
  has_restriction=$(echo "$json" | jq '.view.restriction // null | type == "object" and has("type")')
  if [[ "$has_restriction" == "true" ]]; then
    local r_type r_id
    r_type=$(echo "$json" | jq -r '.view.restriction.type')
    r_id=$(echo "$json" | jq -r '.view.restriction.id')
    tf+="\n  restriction = {\n"
    tf+="    type = \"${r_type}\"\n"
    tf+="    id   = ${r_id}\n"
    tf+="  }\n"
  fi

  tf+="}"

  echo -e "$tf"
}
