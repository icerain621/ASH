#!/usr/bin/env bash
# H-04/H-05 live Worker smoke against real GitHub (explicitly NOT ASH_CI_FIXTURE).
# Opt-in: ASH_CI_LIVE=1. Skips cleanly when env/token missing (exit 0 + SKIP class).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
# shellcheck source=_json_pick.sh
source "$ROOT/scripts/_json_pick.sh"

BASE="${ASH_WORKER_URL:-}"
SPACE="${ASH_SPACE_ID:-local}"
AUTH_HEADER="${ASH_AUTH_HEADER:-}"
OWNER="${ASH_GITHUB_OWNER:-}"
REPO="${ASH_GITHUB_REPO:-}"
TOKEN="${ASH_GITHUB_TOKEN:-}"
CONN_ID="${ASH_REPO_CONNECTION_ID:-}"

skip() {
  local class="$1"
  local msg="$2"
  echo "SKIP H-04/H-05 ci-live: ${class} — ${msg}"
  exit 0
}

fail() {
  echo "FAIL H-04/H-05 ci-live: $*" >&2
  exit 1
}

if [[ "${ASH_CI_LIVE:-}" != "1" ]]; then
  skip "not_enabled" "export ASH_CI_LIVE=1 to run real GitHub sync/diagnose"
fi
if [[ -z "$BASE" ]]; then
  skip "no_worker" "set ASH_WORKER_URL to a running Worker"
fi

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

echo "== CI live smoke (real GitHub) @ ${BASE} space=${SPACE} =="

READYZ=$(curl -sS "${BASE}/readyz" || true)
if echo "$READYZ" | grep -q 'ASH_CI_FIXTURE'; then
  fail "Worker advertises ASH_CI_FIXTURE; restart Worker without fixture for H-04/H-05 live"
fi

if [[ -z "$CONN_ID" ]]; then
  if [[ -z "$TOKEN" || -z "$OWNER" || -z "$REPO" ]]; then
    skip "no_credentials" "set ASH_REPO_CONNECTION_ID or ASH_GITHUB_TOKEN+ASH_GITHUB_OWNER+ASH_GITHUB_REPO"
  fi
  echo "== create secret + repo connection =="
  SECRET_JSON=$(curl_json POST /api/v1/secrets \
    "{\"name\":\"GITHUB_TOKEN_LIVE\",\"value\":\"${TOKEN}\",\"scope\":{\"provider\":\"github\"}}")
  SECRET_ID=$(echo "$SECRET_JSON" | sed -n 's/.*"id"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -1)
  [[ -n "$SECRET_ID" ]] || fail "secret create failed: $SECRET_JSON"
  CONN_JSON=$(curl_json POST /api/v1/repo/connections \
    "{\"provider\":\"github\",\"owner\":\"${OWNER}\",\"repo\":\"${REPO}\",\"secretId\":\"${SECRET_ID}\"}")
  CONN_ID=$(echo "$CONN_JSON" | sed -n 's/.*"id"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -1)
  [[ -n "$CONN_ID" ]] || fail "connection create failed: $CONN_JSON"
fi

echo "== H-04 sync runs connectionId=${CONN_ID} =="
RUNS_JSON=$(curl_json GET "/api/v1/ci/runs?connectionId=${CONN_ID}&sync=true&limit=20")
if echo "$RUNS_JSON" | grep -q 'fixture-run'; then
  fail "unexpected fixture data in live mode"
fi
if echo "$RUNS_JSON" | grep -qE '"code"[[:space:]]*:[[:space:]]*"(CI_|REPO_|PLAINTEXT)'; then
  fail "runs sync failed: $RUNS_JSON"
fi
RUN_ID=$(echo "$RUNS_JSON" | sed -n 's/.*"id"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -1)
if [[ -z "$RUN_ID" ]]; then
  skip "empty_runs" "sync reached API but no workflow runs returned for connection"
fi
echo "runId=${RUN_ID}"

echo "== H-05 sync jobs =="
JOBS_JSON=$(curl_json GET "/api/v1/ci/jobs?runId=${RUN_ID}&sync=true")
if echo "$JOBS_JSON" | grep -q 'fixture-job'; then
  fail "unexpected fixture jobs in live mode"
fi
JOB_ID=$(echo "$JOBS_JSON" | sed -n 's/.*"id"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -1)
if [[ -z "$JOB_ID" ]]; then
  echo "WARN: no jobs for run; diagnose with logText fallback"
  DIAG_JSON=$(curl_json POST /api/v1/ci/failures/diagnose \
    "{\"connectionId\":\"${CONN_ID}\",\"runId\":\"${RUN_ID}\",\"logText\":\"go test ./...\\n--- FAIL: TestLive (0.01s)\\nFAIL\\tpkg\\t0.1s\"}")
else
  echo "jobId=${JOB_ID}"
  echo "== H-05 diagnose by jobId =="
  DIAG_JSON=$(curl_json POST /api/v1/ci/failures/diagnose "{\"jobId\":\"${JOB_ID}\"}")
fi

if ! echo "$DIAG_JSON" | grep -q '"rootCause"'; then
  fail "diagnose failed: $DIAG_JSON"
fi
ROOT_CAUSE=$(echo "$DIAG_JSON" | sed -n 's/.*"rootCause"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -1)
echo "rootCause=${ROOT_CAUSE}"
echo "$DIAG_JSON" | head -c 320
echo ""
echo "OK CI live H-04/H-05 (real GitHub)"
