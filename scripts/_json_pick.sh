#!/usr/bin/env bash
# Pick item id from JSON list by matching a provider/key field (bash-only).

json_pick_field() {
  local json="$1"
  local match_value="$2"
  local match_key="${3:-providerJobId}"
  local id_key="${4:-id}"
  local block
  block=$(printf '%s' "$json" | tr ',' '\n' | grep -F "\"${match_key}\":\"${match_value}\"" | head -1 || true)
  if [[ -z "$block" ]]; then
    block=$(printf '%s' "$json" | grep -o "{[^}]*\"${match_key}\":\"${match_value}\"[^}]*}" | head -1 || true)
  fi
  if [[ -z "$block" ]]; then
    return 1
  fi
  local item="${block}"
  if [[ "$item" != *"\"${id_key}\""* ]]; then
    item=$(printf '%s' "$json" | sed 's/{"items":\[//;s/\]}$//' | tr '}' '\n' | grep -F "\"${match_key}\":\"${match_value}\"" | head -1)
  fi
  printf '%s' "$item" | sed -n "s/.*\"${id_key}\":\"\([^\"]*\)\".*/\1/p" | head -1
}
