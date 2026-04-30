#!/usr/bin/env bash
# Generator: zendesk_custom_role
# Maps Zendesk API /api/v2/custom_roles/:id response to Terraform resource

# Emit a single configuration attribute if present in the JSON.
# $1: json; $2: attribute path (e.g. .custom_role.configuration.chat_access)
# $3: terraform attribute name; $4: type (bool or string)
_emit_cfg_attr() {
  local json="$1" jq_path="$2" tf_name="$3" tf_type="$4"
  local val
  val=$(echo "$json" | jq -r "${jq_path} // empty")
  if [[ -z "$val" || "$val" == "null" ]]; then
    return
  fi
  if [[ "$tf_type" == "bool" ]]; then
    echo "    ${tf_name} = ${val}"
  else
    echo "    ${tf_name} = \"${val}\""
  fi
}

generate_custom_role() {
  local json="$1"
  local name description role_type res_name

  name=$(echo "$json" | jq -r '.custom_role.name')
  description=$(echo "$json" | jq -r '.custom_role.description // ""')
  role_type=$(echo "$json" | jq -r '.custom_role.role_type // 0')
  res_name=$(to_snake_case "$name")

  local tf=""
  tf+="resource \"zendesk_custom_role\" \"${res_name}\" {\n"
  tf+="  name        = \"${name}\"\n"
  if [[ -n "$description" ]]; then
    tf+="  description = \"${description}\"\n"
  fi
  tf+="  role_type   = ${role_type}\n"

  local has_cfg
  has_cfg=$(echo "$json" | jq '.custom_role.configuration // null | type == "object"')
  if [[ "$has_cfg" == "true" ]]; then
    tf+="\n  configuration = {\n"

    # Boolean fields. The following are read-only on the Zendesk API and the
    # provider exposes them as Computed-only attributes — emitting them in
    # config would produce "Cannot set value for this attribute" errors:
    #   assign_tickets_to_any_group, chat_access, group_access, light_agent,
    #   moderate_forums, organization_notes_editing.
    local bool_fields=(
      "manage_business_rules"
      "manage_contextual_workspaces"
      "manage_dynamic_content"
      "manage_extensions_and_channels"
      "manage_facebook"
      "manage_organization_fields"
      "manage_ticket_fields"
      "manage_ticket_forms"
      "manage_user_fields"
      "organization_editing"
      "side_conversation_create"
      "ticket_deletion"
      "ticket_editing"
      "ticket_merge"
      "ticket_tag_editing"
      "view_deleted_tickets"
      "voice_access"
      "voice_dashboard_access"
    )
    for field in "${bool_fields[@]}"; do
      local line
      line=$(_emit_cfg_attr "$json" ".custom_role.configuration.${field}" "$field" "bool")
      if [[ -n "$line" ]]; then
        tf+="${line}\n"
      fi
    done

    # String fields
    local str_fields=(
      "end_user_list_access"
      "end_user_profile_access"
      "explore_access"
      "forum_access"
      "macro_access"
      "report_access"
      "ticket_access"
      "ticket_comment_access"
      "view_access"
    )
    for field in "${str_fields[@]}"; do
      local line
      line=$(_emit_cfg_attr "$json" ".custom_role.configuration.${field}" "$field" "string")
      if [[ -n "$line" ]]; then
        tf+="${line}\n"
      fi
    done

    tf+="  }\n"
  fi

  tf+="}"

  echo -e "$tf"
}
