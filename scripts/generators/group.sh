#!/usr/bin/env bash
# Generator: zendesk_group
# Maps Zendesk API /api/v2/groups/:id response to Terraform resource

generate_group() {
  local json="$1"
  local name description is_public res_name

  name=$(echo "$json" | jq -r '.group.name')
  description=$(echo "$json" | jq -r '.group.description // ""')
  is_public=$(jq_bool "$json" '.group.is_public' true)
  res_name=$(to_snake_case "$name")

  cat <<EOTF
resource "zendesk_group" "${res_name}" {
  name        = "${name}"
  description = "${description}"
  is_public   = ${is_public}
}
EOTF
}

# Generate the zendesk_group_membership resource block.
# $1: JSON array of [{user_name, user_id, membership_id}]
# $2: group resource name (snake_case)
generate_group_membership() {
  local members_json="$1"
  local group_res_name="$2"
  local count

  count=$(echo "$members_json" | jq 'length')
  if [[ "$count" -eq 0 ]]; then
    return
  fi

  local tf=""
  tf+="\nresource \"zendesk_group_membership\" \"${group_res_name}\" {\n"
  tf+="  for_each = toset([\n"
  for i in $(seq 0 $((count - 1))); do
    local user_name data_name
    user_name=$(echo "$members_json" | jq -r ".[$i].user_name")
    data_name=$(to_snake_case "$user_name")
    tf+="    data.zendesk_user.${data_name}.id,\n"
  done
  tf+="  ])\n"
  tf+="\n"
  tf+="  user_id  = each.value\n"
  tf+="  group_id = zendesk_group.${group_res_name}.id\n"
  tf+="}"

  echo -e "$tf"
}

# Generate data "zendesk_user" blocks for members not already in users.tf.
# $1: JSON array of [{user_name, user_id, membership_id}]
# $2: path to existing users.tf
generate_user_data_sources() {
  local members_json="$1"
  local users_tf="$2"
  local count

  count=$(echo "$members_json" | jq 'length')
  if [[ "$count" -eq 0 ]]; then
    return
  fi

  local tf=""
  for i in $(seq 0 $((count - 1))); do
    local user_id user_name data_name
    user_id=$(echo "$members_json" | jq -r ".[$i].user_id")
    user_name=$(echo "$members_json" | jq -r ".[$i].user_name")
    data_name=$(to_snake_case "$user_name")

    # Skip if this data source already exists in users.tf
    if [[ -f "$users_tf" ]] && grep -q "\"${data_name}\"" "$users_tf"; then
      echo "  (skipping data.zendesk_user.${data_name} — already exists)" >&2
      continue
    fi

    tf+="\ndata \"zendesk_user\" \"${data_name}\" {\n"
    tf+="  id = \"${user_id}\"\n"
    tf+="}\n"
  done

  echo -e "$tf"
}
