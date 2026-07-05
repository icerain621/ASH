#!/usr/bin/env bash
# H-09 business sampling against a running Worker (postgres-rds-e2e.md §7).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

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

echo "== H-09 release sampling @ ${BASE} space=${SPACE} =="

echo "== 7.0 readyz =="
curl_json GET /readyz | head -c 400
echo ""

echo "== 7.1 create run =="
RUN_JSON=$(curl_json POST /api/v1/runs '{"scenario":{"name":"feature_delivery","scenarioVersion":"1.0.0"},"inputs":{"issueOrSpec":"h09 smoke","repoRoot":"."}}')
RUN_ID=$(echo "$RUN_JSON" | sed -n 's/.*"runId"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -1)
if [[ -z "$RUN_ID" ]]; then
  echo "create run failed: $RUN_JSON" >&2
  exit 1
fi
echo "runId=$RUN_ID"

echo "== 7.2 SSE stream (live) =="
echo "Tip: GET ${BASE}/api/v1/runs/${RUN_ID}/stream (manual); CI uses TestReleaseSamplingSSE"

echo "== 7.3 memory candidate + review + query =="
CAND=$(curl_json POST /api/v1/memory/candidates '{"layer":"L1","title":"H09 live","body":"release sampling","scopeRepo":"ash","evidence":[{"kind":"file","ref":"doc/h09.md"}]}')
CID=$(echo "$CAND" | sed -n 's/.*"candidateId"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -1)
if [[ -z "$CID" ]]; then
  echo "candidate failed: $CAND" >&2
  exit 1
fi
curl_json POST "/api/v1/memory/candidates/${CID}/review" '{"decision":"approve","reason":"h09","policyProfile":"default"}' >/dev/null
curl_json POST /api/v1/memory/query '{"text":"H09 live","topK":3}' | head -c 200
echo ""

echo "== 7.3b memory ttl-queue =="
curl_json GET /api/v1/memory/ttl-queue?limit=5 | head -c 200
echo ""

echo "== 7.4 metrics overview =="
curl_json GET "/api/v1/metrics/overview?spaceId=${SPACE}" | head -c 200
echo ""

echo "== 7.5 ci diagnose =="
curl_json POST /api/v1/ci/failures/diagnose '{"logText":"go test ./...\n--- FAIL: TestH09\nFAIL\tpkg\t0.1s"}' | head -c 200
echo ""
echo "Tip: Worker ASH_CI_FIXTURE=1 → bash scripts/ci-fixture-smoke.sh (H-04/H-05 jobId 诊断)"

echo "== 7.6 compliance export =="
curl_json POST /api/v1/compliance/export '{"suite":"TR2"}' | head -c 200
echo ""

echo "== 7.7 scale readiness =="
curl_json GET /api/v1/scale/readiness | head -c 300
echo ""

echo "OK H-09 sampling (7.2 SSE manual: GET /api/v1/runs/${RUN_ID}/stream)"
