#!/usr/bin/env bash
# H-04/H-05 live Worker smoke when ASH_CI_FIXTURE=1 (no GitHub API).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

# shellcheck source=_json_pick.sh
source "$ROOT/scripts/_json_pick.sh"

BASE="${ASH_WORKER_URL:-http://127.0.0.1:8080}"
SPACE="${ASH_SPACE_ID:-local}"
AUTH_HEADER="${ASH_AUTH_HEADER:-}"

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

echo "== CI fixture smoke @ ${BASE} space=${SPACE} =="

READYZ=$(curl_json GET /readyz)
if ! echo "$READYZ" | grep -q 'ASH_CI_FIXTURE'; then
  echo "Worker readyz missing ASH_CI_FIXTURE liveGateHint; start worker with ASH_CI_FIXTURE=1" >&2
  echo "$READYZ" | head -c 400 >&2
  exit 1
fi

echo "== create secret + repo connection =="
SECRET_JSON=$(curl_json POST /api/v1/secrets '{"name":"GITHUB_TOKEN_FIXTURE","value":"ghp_fixture","scope":{"provider":"github"}}')
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

echo "== H-04 sync runs =="
RUNS_JSON=$(curl_json GET "/api/v1/ci/runs?connectionId=${CONN_ID}&sync=true")
if ! echo "$RUNS_JSON" | grep -q 'fixture-run-9001'; then
  echo "runs sync failed: $RUNS_JSON" >&2
  exit 1
fi
RUN_ID=$(json_pick_field "$RUNS_JSON" "fixture-run-9001" "providerRunId" "id")
if [[ -z "$RUN_ID" ]]; then
  RUN_ID=$(echo "$RUNS_JSON" | sed -n 's/.*"id"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -1)
fi

echo "== H-05 sync jobs =="
JOBS_JSON=$(curl_json GET "/api/v1/ci/jobs?runId=${RUN_ID}&sync=true")
if ! echo "$JOBS_JSON" | grep -q 'fixture-job-9101'; then
  echo "jobs sync failed: $JOBS_JSON" >&2
  exit 1
fi
JOB_ID=$(json_pick_field "$JOBS_JSON" "fixture-job-9101" "providerJobId" "id")
if [[ -z "$JOB_ID" ]]; then
  JOB_ID=$(echo "$JOBS_JSON" | sed -n 's/.*"id"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -1)
fi

echo "== H-05 diagnose by jobId (fetch fixture logs) =="
DIAG_JSON=$(curl_json POST /api/v1/ci/failures/diagnose "{\"jobId\":\"${JOB_ID}\"}")
if ! echo "$DIAG_JSON" | grep -q 'test_failure'; then
  echo "diagnose failed: $DIAG_JSON" >&2
  exit 1
fi
echo "$DIAG_JSON" | head -c 240
echo ""
echo "OK CI fixture H-04/H-05"
