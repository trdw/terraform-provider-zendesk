#!/usr/bin/env bash
# Generator: zendesk_user
# Maps Zendesk API /api/v2/users/:id response to Terraform resource

_emit_user_str() {
  local json="$1" jq_path="$2" tf_name="$3"
  local val
  val=$(echo "$json" | jq -r "${jq_path} // empty")
  if [[ -n "$val" && "$val" != "null" ]]; then
    echo "  ${tf_name} = \"${val}\""
  fi
}

_emit_user_int() {
  local json="$1" jq_path="$2" tf_name="$3"
  local val
  val=$(echo "$json" | jq -r "${jq_path} // empty")
  if [[ -n "$val" && "$val" != "null" ]]; then
    echo "  ${tf_name} = ${val}"
  fi
}

generate_user() {
  local json="$1"
  local name email role res_name

  name=$(echo "$json" | jq -r '.user.name')
  email=$(echo "$json" | jq -r '.user.email')
  role=$(echo "$json" | jq -r '.user.role // "end-user"')
  res_name=$(to_snake_case "$name")

  if [[ "$role" == "end-user" ]]; then
    echo "Warning: user '${name}' has role 'end-user'; zendesk_user is typically used for agents." >&2
  fi

  local tf=""
  tf+="resource \"zendesk_user\" \"${res_name}\" {\n"
  tf+="  name  = \"${name}\"\n"
  tf+="  email = \"${email}\"\n"
  tf+="  role  = \"${role}\"\n"

  local line
  line=$(_emit_user_int "$json" ".user.custom_role_id"   "custom_role_id");   [[ -n "$line" ]] && tf+="${line}\n"
  line=$(_emit_user_int "$json" ".user.default_group_id" "default_group_id"); [[ -n "$line" ]] && tf+="${line}\n"
  line=$(_emit_user_str "$json" ".user.alias"            "alias");            [[ -n "$line" ]] && tf+="${line}\n"
  line=$(_emit_user_str "$json" ".user.signature"        "signature");        [[ -n "$line" ]] && tf+="${line}\n"
  line=$(_emit_user_str "$json" ".user.phone"            "phone");            [[ -n "$line" ]] && tf+="${line}\n"
  line=$(_emit_user_str "$json" ".user.locale"           "locale");           [[ -n "$line" ]] && tf+="${line}\n"
  line=$(_emit_user_str "$json" ".user.time_zone"        "time_zone");        [[ -n "$line" ]] && tf+="${line}\n"
  line=$(_emit_user_str "$json" ".user.ticket_restriction" "ticket_restriction"); [[ -n "$line" ]] && tf+="${line}\n"
  line=$(_emit_user_int "$json" ".user.organization_id"  "organization_id");  [[ -n "$line" ]] && tf+="${line}\n"
  line=$(_emit_user_str "$json" ".user.external_id"      "external_id");      [[ -n "$line" ]] && tf+="${line}\n"

  local suspended
  suspended=$(echo "$json" | jq -r '.user.suspended // false')
  if [[ "$suspended" == "true" ]]; then
    tf+="  suspended = true\n"
  fi

  tf+="}"

  echo -e "$tf"
}
