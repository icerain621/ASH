#!/usr/bin/env bash
# Waker smoke (Sprint DX6 / DX12): package tests + optional live Worker queue/status.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
# shellcheck source=_go_env.sh
source "$ROOT/scripts/_go_env.sh"
_ash_go_env_bootstrap "$ROOT"

echo "== waker package + API tests =="
go test ./internal/waker/ ./internal/api/ -count=1 -run 'TestWaker|TestEnsure|TestRunDue|TestQueue|TestRunDoctor|TestRunKPI|TestParse|TestStatusAuto|TestSetDuty|TestProbes'

BASE="${ASH_WORKER_URL:-}"
if [[ -n "$BASE" ]]; then
  echo "== live waker queue @ ${BASE} =="
  curl -sf "${BASE}/api/v1/waker/queue?limit=5" | head -c 400
  echo ""
  echo "== live waker status @ ${BASE} =="
  curl -sf "${BASE}/api/v1/waker/status?spaceId=local&recent=3" | head -c 500
  echo ""
else
  echo "== live Worker skipped (set ASH_WORKER_URL) =="
fi

echo "OK waker-smoke"
