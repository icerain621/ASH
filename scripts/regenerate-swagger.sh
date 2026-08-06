#!/usr/bin/env bash
# Regenerate OpenAPI docs (Git Bash / Linux / macOS)
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
# shellcheck source=_go_env.sh
source "$ROOT/scripts/_go_env.sh"
# shellcheck source=_swag.sh
source "$ROOT/scripts/_swag.sh"
_ash_go_env_bootstrap "$ROOT"

swag_err="$(mktemp)"
trap 'rm -f "$swag_err"' EXIT
if ! _ash_swag_init "$ROOT" > /dev/null 2>"$swag_err"; then
  if grep -qiE 'sumdb|sum\.golang\.org|proxy\.golang\.org|connectex|connection.*failed|i/o timeout' "$swag_err"; then
    echo "warning: module proxy/checksum unreachable; retrying swag (GOSUMDB=off, GOPROXY mirror)"
    ASH_GOSUMDB_OFF=1 GOPROXY="${GOPROXY:-https://goproxy.cn,direct}" _ash_swag_init "$ROOT"
  else
    cat "$swag_err" >&2
    exit 1
  fi
fi

echo "Swagger written to internal/api/docs/"
