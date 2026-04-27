#!/usr/bin/env bash
# Generator: zendesk_ticket_form
# Maps Zendesk API /api/v2/ticket_forms/:id response to Terraform resource

generate_ticket_form() {
  # $2 (optional): JSON object mapping ticket-field ID (string) to its full
  # Terraform address (without the `.id` suffix). For example,
  #   {"123":"zendesk_ticket_field.vin",
  #    "1":"data.zendesk_ticket_field.subject"}
  # produces `zendesk_ticket_field.vin.id` and `data.zendesk_ticket_field.subject.id`
  # respectively. IDs not in the map are emitted as raw integers.
  # $3 (optional): same shape, but for brand IDs in restricted_brand_ids.
  local json="$1"
  local field_id_map="${2:-}"
  local brand_id_map="${3:-}"
  [[ -z "$field_id_map" ]] && field_id_map='{}'
  [[ -z "$brand_id_map" ]] && brand_id_map='{}'
  local name display_name active default end_user_visible position in_all_brands res_name

  name=$(echo "$json" | jq -r '.ticket_form.name')
  display_name=$(echo "$json" | jq -r '.ticket_form.display_name // empty')
  active=$(jq_bool "$json" '.ticket_form.active' true)
  default=$(jq_bool "$json" '.ticket_form.default' false)
  end_user_visible=$(jq_bool "$json" '.ticket_form.end_user_visible' true)
  # position is read-only in the provider — Zendesk reshuffles it on create,
  # so the field is not emitted here. Use the dedicated reorder endpoint to
  # change form order.
  in_all_brands=$(jq_bool "$json" '.ticket_form.in_all_brands' false)
  res_name=$(to_snake_case "$name")

  local tf=""
  tf+="resource \"zendesk_ticket_form\" \"${res_name}\" {\n"
  tf+="  name             = \"${name}\"\n"
  if [[ -n "$display_name" ]]; then
    tf+="  display_name     = \"${display_name}\"\n"
  fi
  tf+="  active           = ${active}\n"
  tf+="  default          = ${default}\n"
  tf+="  end_user_visible = ${end_user_visible}\n"
  tf+="  in_all_brands    = ${in_all_brands}\n"

  # ticket_field_ids — emit `<addr>.id` for any ID in field_id_map, raw
  # integer otherwise.
  local field_id_count
  field_id_count=$(echo "$json" | jq '[.ticket_form.ticket_field_ids[]?] | length')
  if [[ "$field_id_count" -gt 0 ]]; then
    local field_refs=()
    local i fid addr
    for i in $(seq 0 $((field_id_count - 1))); do
      fid=$(echo "$json" | jq -r ".ticket_form.ticket_field_ids[$i]")
      addr=$(echo "$field_id_map" | jq -r --arg id "$fid" '.[$id] // empty')
      if [[ -n "$addr" ]]; then
        field_refs+=("${addr}.id")
      else
        field_refs+=("$fid")
      fi
    done
    local joined
    joined=$(printf ', %s' "${field_refs[@]}")
    joined="${joined:2}"
    tf+="\n  ticket_field_ids = [${joined}]\n"
  fi

  # restricted_brand_ids — emit `<addr>.id` for any ID in brand_id_map, raw
  # integer otherwise.
  local brand_id_count
  brand_id_count=$(echo "$json" | jq '[.ticket_form.restricted_brand_ids[]?] | length')
  if [[ "$brand_id_count" -gt 0 ]]; then
    local brand_refs=()
    local j bid baddr
    for j in $(seq 0 $((brand_id_count - 1))); do
      bid=$(echo "$json" | jq -r ".ticket_form.restricted_brand_ids[$j]")
      baddr=$(echo "$brand_id_map" | jq -r --arg id "$bid" '.[$id] // empty')
      if [[ -n "$baddr" ]]; then
        brand_refs+=("${baddr}.id")
      else
        brand_refs+=("$bid")
      fi
    done
    local bjoined
    bjoined=$(printf ', %s' "${brand_refs[@]}")
    bjoined="${bjoined:2}"
    tf+="  restricted_brand_ids = [${bjoined}]\n"
  fi

  tf+="}"

  echo -e "$tf"
}
