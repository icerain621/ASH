#!/usr/bin/env bash
# Pre-release local readiness: fast §8 bundle + live Worker smokes (no cloud required).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

echo "== release-window-gate (fast) =="
bash scripts/release-window-gate.sh

echo "== worker-local-gate =="
bash scripts/worker-local-gate.sh

if [[ "${ASH_LOCAL_READINESS_JWT:-1}" == "1" ]]; then
  echo "== worker-production-gate =="
  bash scripts/worker-production-gate.sh
else
  echo "skip worker-production-gate (ASH_LOCAL_READINESS_JWT=0)"
fi

echo "OK local-readiness-gate"
