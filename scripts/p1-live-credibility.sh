#!/usr/bin/env bash
# P1 credibility orchestrator: real CI live (H-04/05) + ExecGo live (H-06).
# Missing env → SKIP (exit 0). Hard failures in an enabled track → exit 1.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

echo "== P1 live credibility =="
FAIL=0

echo ""
echo "-- Track H-04/H-05 (ci-live-smoke) --"
set +e
bash scripts/ci-live-smoke.sh
CI_CODE=$?
set -e
if [[ "$CI_CODE" -ne 0 ]]; then
  FAIL=1
fi

echo ""
echo "-- Track H-06 (execgo-live-smoke) --"
if [[ "${ASH_EXECGO_E2E:-}" != "1" ]]; then
  echo "SKIP H-06: set ASH_EXECGO_E2E=1 (and ExecGo/Codex) to run live M3-05"
else
  set +e
  bash scripts/execgo-live-smoke.sh
  EG_CODE=$?
  set -e
  if [[ "$EG_CODE" -ne 0 ]]; then
    FAIL=1
  fi
fi

echo ""
if [[ "$FAIL" -ne 0 ]]; then
  echo "FAIL p1-live-credibility — see doc/checklists/p1-live-credibility.md" >&2
  exit 1
fi
echo "OK p1-live-credibility (enabled tracks passed or skipped)"
