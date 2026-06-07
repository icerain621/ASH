# shellcheck shell=bash
# Source after _go_env.sh. Runs swag init with a pinned module version (see go.mod).

_ash_swag_init() {
  local root="${1:?root required}"
  shift || true
  local version="${ASH_SWAG_VERSION:-v1.16.4}"
  local -a args=(
    init
    -g cmd/worker/main.go
    -o internal/api/docs
    --parseDependency
    --parseInternal
  )
  if [[ $# -gt 0 ]]; then
    args+=("$@")
  fi
  if [[ "${ASH_SWAG_USE_PATH:-}" == "1" ]] && command -v swag >/dev/null 2>&1; then
    (cd "$root" && swag "${args[@]}")
    return
  fi
  local sumdb="${GOSUMDB:-}"
  if [[ "${ASH_GOSUMDB_OFF:-}" == "1" ]]; then
    sumdb=off
  fi
  (cd "$root" && GOSUMDB="$sumdb" go run "github.com/swaggo/swag/cmd/swag@${version}" "${args[@]}")
}
