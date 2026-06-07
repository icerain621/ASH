#!/usr/bin/env bash
# OpenAPI checks: swag regen determinism + contract alignment.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
# shellcheck source=_go_env.sh
source "$ROOT/scripts/_go_env.sh"
# shellcheck source=_swag.sh
source "$ROOT/scripts/_swag.sh"
_ash_go_env_bootstrap "$ROOT"

echo "== openapi: contract alignment (/api/v1 in doc/api/openapi-ash-v1.yaml) =="
go test ./internal/openapicheck -run TestContractMatchesSwagger -count=1

echo "== api: error code catalog parity =="
go test ./internal/apicodes -count=1

echo "== openapi: swag regen determinism =="
tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT
cp -R internal/api/docs "$tmpdir/docs"
swag_err="$(mktemp)"
trap 'rm -rf "$tmpdir" "$swag_err"' EXIT
if ! _ash_swag_init "$ROOT" > /dev/null 2>"$swag_err"; then
  if grep -qiE 'sumdb|sum\.golang\.org|proxy\.golang\.org|connectex|connection.*failed|i/o timeout' "$swag_err"; then
    echo "warning: module proxy/checksum unreachable; retrying swag (GOSUMDB=off, GOPROXY mirror)"
    ASH_GOSUMDB_OFF=1 GOPROXY="${GOPROXY:-https://goproxy.cn,direct}" _ash_swag_init "$ROOT" > /dev/null
  else
    cat "$swag_err" >&2
    exit 1
  fi
fi
diff -ru "$tmpdir/docs" internal/api/docs

echo "openapi-check OK"
