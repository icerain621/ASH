#!/usr/bin/env bash
# Live Worker smoke: H-04/H-05/H-06/H-07/H-09 (requires ASH_WORKER_URL).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

BASE="${ASH_WORKER_URL:-}"
if [[ -z "$BASE" ]]; then
  echo "ASH_WORKER_URL is required for live smoke" >&2
  exit 1
fi

echo "== live smoke @ ${BASE} =="

echo "== readyz =="
if ! curl -sf "${BASE}/readyz" | head -c 400; then
  echo "readyz failed @ ${BASE}" >&2
  exit 1
fi
echo ""

echo "== H-09 release sampling =="
bash scripts/release-sampling.sh

if curl -sf "${BASE}/readyz" | grep -q 'ASH_CI_FIXTURE'; then
  echo "== H-04/H-05 CI fixture smoke =="
  bash scripts/ci-fixture-smoke.sh
  echo "== H-07 secret rotate smoke =="
  ASH_CI_FIXTURE=1 bash scripts/secret-rotate-smoke.sh
else
  echo "== H-04/H-05/H-07 fixture smokes skipped (Worker without ASH_CI_FIXTURE) =="
fi

if [[ "${ASH_EXECGO_E2E:-}" == "1" ]]; then
  echo "== H-06 ExecGo live smoke =="
  bash scripts/execgo-live-smoke.sh
else
  echo "== H-06 ExecGo live smoke skipped (set ASH_EXECGO_E2E=1) =="
fi

echo "OK live smoke"
