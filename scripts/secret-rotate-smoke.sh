#!/usr/bin/env bash
# H-07 secret rotate smoke: create secret + repo connection, rotate, re-sync CI (fixture-friendly).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

BASE="${ASH_WORKER_URL:-http://127.0.0.1:8080}"
SPACE="${ASH_SPACE_ID:-local}"
AUTH_HEADER="${ASH_AUTH_HEADER:-}"
FIXTURE="${ASH_CI_FIXTURE:-0}"

curl_json() {
  local method="$1"
  local path="$2"
  local body="${3:-}"
  local args=(-sS -X "$method" "${BASE}${path}" -H "X-ASH-Space-ID: ${SPACE}")
  if [[ -n "$AUTH_HEADER" ]]; then
    args+=(-H "Authorization: ${AUTH_HEADER}")
  fi
  if [[ -n "$body" ]]; then
    args+=(-H "Content-Type: application/json" -d "$body")
  fi
  curl "${args[@]}"
}

echo "== H-07 secret rotate smoke @ ${BASE} space=${SPACE} =="

if [[ "$FIXTURE" == "1" ]]; then
  READYZ=$(curl_json GET /readyz)
  if ! echo "$READYZ" | grep -q 'ASH_CI_FIXTURE'; then
    echo "WARN: ASH_CI_FIXTURE=1 but Worker readyz missing hint; CI sync may call GitHub API" >&2
  fi
fi

echo "== create secret + repo connection =="
SECRET_JSON=$(curl_json POST /api/v1/secrets '{"name":"GITHUB_TOKEN_ROTATE","value":"ghp_before","scope":{"provider":"github"}}')
SECRET_ID=$(echo "$SECRET_JSON" | sed -n 's/.*"id"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -1)
if [[ -z "$SECRET_ID" ]]; then
  echo "secret create failed: $SECRET_JSON" >&2
  exit 1
fi
CONN_JSON=$(curl_json POST /api/v1/repo/connections "{\"provider\":\"github\",\"owner\":\"iammm0\",\"repo\":\"ASH\",\"secretId\":\"${SECRET_ID}\"}")
CONN_ID=$(echo "$CONN_JSON" | sed -n 's/.*"id"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -1)
if [[ -z "$CONN_ID" ]]; then
  echo "connection create failed: $CONN_JSON" >&2
  exit 1
fi

echo "== pre-rotate CI sync =="
RUNS_JSON=$(curl_json GET "/api/v1/ci/runs?connectionId=${CONN_ID}&sync=true")
if [[ "$FIXTURE" == "1" ]] && ! echo "$RUNS_JSON" | grep -q 'fixture-run-9001'; then
  echo "pre-rotate sync failed: $RUNS_JSON" >&2
  exit 1
fi

echo "== rotate secret =="
ROTATE_JSON=$(curl_json POST "/api/v1/secrets/${SECRET_ID}/rotate" '{"value":"ghp_after"}')
if ! echo "$ROTATE_JSON" | grep -q '"id"'; then
  echo "rotate failed: $ROTATE_JSON" >&2
  exit 1
fi

echo "== post-rotate CI sync =="
RUNS_AFTER=$(curl_json GET "/api/v1/ci/runs?connectionId=${CONN_ID}&sync=true")
if [[ "$FIXTURE" == "1" ]] && ! echo "$RUNS_AFTER" | grep -q 'fixture-run-9001'; then
  echo "post-rotate sync failed: $RUNS_AFTER" >&2
  exit 1
fi

echo "OK H-07 secret rotate smoke"
