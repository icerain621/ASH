#!/usr/bin/env bash
# Sprint DH: Harness Profile unit/API smoke (no live Worker required).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
# shellcheck source=_go_env.sh
source "$ROOT/scripts/_go_env.sh"
_ash_go_env_bootstrap "$ROOT"

echo "== harness service + API tests =="
go test ./internal/harness/ -count=1
go test ./internal/harness/loop/ ./internal/sandbox/ -count=1
go test ./internal/api/ -run 'TestHarnessProfileAPILifecycle' -count=1
go test ./internal/store/ -run 'TestRLSExpectedPolicyCount|TestVerifyRLSMigrationSQL|TestMigrationCatalog_RLSCoverage' -count=1
go test ./internal/runs/ -run 'TestHarnessLoopEmitsRoutedAndCompleted' -count=1

BASE="${ASH_WORKER_URL:-}"
if [[ -n "$BASE" ]]; then
  SPACE="${ASH_SPACE_ID:-local}"
  AUTH_HEADER="${ASH_AUTH_HEADER:-}"
  echo "== live harness profile @ ${BASE} space=${SPACE} =="
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
  CREATE=$(curl_json POST /api/v1/harness/profiles '{"name":"smoke-default","spec":{"provider":{"kind":"static"},"sandbox":{"defaultMode":"workspace-write","network":"deny","spillMaxBytes":65536},"tools":{"allowlist":["read","write"]},"policyProfile":"default"}}')
  ID=$(echo "$CREATE" | sed -n 's/.*"id"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -1)
  if [[ -z "$ID" ]]; then
    echo "create failed: $CREATE" >&2
    exit 1
  fi
  PROMOTE=$(curl_json POST "/api/v1/harness/profiles/${ID}/promote")
  if ! echo "$PROMOTE" | grep -q '"status"[[:space:]]*:[[:space:]]*"active"'; then
    echo "promote failed: $PROMOTE" >&2
    exit 1
  fi
  ACTIVE=$(curl_json GET '/api/v1/harness/profiles/active?name=smoke-default')
  if ! echo "$ACTIVE" | grep -q "$ID"; then
    echo "load active failed: $ACTIVE" >&2
    exit 1
  fi
  echo "live OK id=$ID"
else
  echo "== live harness skipped (set ASH_WORKER_URL) =="
fi

echo "OK harness-smoke"
