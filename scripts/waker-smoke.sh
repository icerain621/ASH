#!/usr/bin/env bash
# Waker smoke (Sprint DX6): package tests + optional live Worker queue.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
# shellcheck source=_go_env.sh
source "$ROOT/scripts/_go_env.sh"
_ash_go_env_bootstrap "$ROOT"

echo "== waker package + API tests =="
go test ./internal/waker/ ./internal/api/ -count=1 -run 'TestWaker|TestQueueAndSweep'

BASE="${ASH_WORKER_URL:-}"
if [[ -n "$BASE" ]]; then
  echo "== live waker queue @ ${BASE} =="
  curl -sf "${BASE}/api/v1/waker/queue?limit=5" | head -c 400
  echo ""
else
  echo "== live Worker skipped (set ASH_WORKER_URL) =="
fi

echo "OK waker-smoke"
