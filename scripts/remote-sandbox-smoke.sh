#!/usr/bin/env bash
# Remote sandbox smoke (Sprint DX40): unit + mock prefer; optional live E2B.
# No cloud key required — live segment skip-pass when unset.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
# shellcheck source=_go_env.sh
source "$ROOT/scripts/_go_env.sh"
_ash_go_env_bootstrap "$ROOT"

EVIDENCE="$ROOT/doc/evidence/remote-sandbox-smoke-latest.md"
STAMP="$(date -u +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || date -u)"
OS_NAME="$(uname -s 2>/dev/null || echo unknown)"
LOG="$(mktemp 2>/dev/null || mktemp -t ash-remote-sbx)"

echo "== remote sandbox unit + router prefer =="
set +e
go test ./internal/sandbox/ ./internal/sandbox/remote/ -count=1 -timeout 120s 2>&1 | tee "$LOG"
RC=${PIPESTATUS[0]}
set -e

STATUS="pass"
if [[ "$RC" -ne 0 ]]; then
  STATUS="fail"
fi

LIVE_STATUS="skipped"
LIVE_NOTE="set ASH_SANDBOX_REMOTE_LIVE=1 + ASH_SANDBOX_REMOTE_API_KEY for live E2B"
if [[ "${ASH_SANDBOX_REMOTE_LIVE:-}" == "1" ]]; then
  if [[ -z "${ASH_SANDBOX_REMOTE_API_KEY:-}" ]]; then
    LIVE_STATUS="skipped"
    LIVE_NOTE="ASH_SANDBOX_REMOTE_LIVE=1 but no ASH_SANDBOX_REMOTE_API_KEY (skip pass)"
    echo "== live E2B: skip (no API key) =="
  else
    echo "== live E2B (ASH_SANDBOX_REMOTE_LIVE=1) =="
    LIVE_LOG="$(mktemp 2>/dev/null || mktemp -t ash-remote-live)"
    set +e
    env ASH_SANDBOX_REMOTE=1 \
      ASH_SANDBOX_REMOTE_BACKEND=e2b \
      go test ./internal/sandbox/remote/ -count=1 -timeout 180s -run 'TestLiveE2B' -v 2>&1 | tee "$LIVE_LOG"
    LIVE_RC=${PIPESTATUS[0]}
    set -e
    if [[ "$LIVE_RC" -eq 0 ]]; then
      LIVE_STATUS="pass"
      LIVE_NOTE="TestLiveE2B ok"
    else
      LIVE_STATUS="fail"
      LIVE_NOTE="TestLiveE2B exit ${LIVE_RC}"
      if [[ "$STATUS" == "pass" ]]; then
        STATUS="fail"
        RC="$LIVE_RC"
      fi
    fi
    {
      echo
      echo "## Live excerpt"
      echo
      echo '```'
      tail -n 40 "$LIVE_LOG" || true
      echo '```'
    } >>"$LOG"
    rm -f "$LIVE_LOG"
  fi
fi

{
  echo "# Remote sandbox smoke evidence (DX40)"
  echo
  echo "| Field | Value |"
  echo "|-------|--------|"
  echo "| Status | **${STATUS}** (exit ${RC}) |"
  echo "| Platform | ${OS_NAME} |"
  echo "| Date | ${STAMP} |"
  echo "| Scope | remote unit / mock / prefer+fallback / e2b httptest |"
  echo "| Live E2B | **${LIVE_STATUS}** — ${LIVE_NOTE} |"
  echo "| Doctor | unchanged ALL 57 / M4 10 (no new case; D5) |"
  echo
  echo "## Env"
  echo
  echo "| Variable | Default / note |"
  echo "|----------|----------------|"
  echo "| \`ASH_SANDBOX_REMOTE\` | off (opt-in prefer for isolated) |"
  echo "| \`ASH_SANDBOX_REMOTE_BACKEND\` | \`mock\` \| \`e2b\` |"
  echo "| \`ASH_SANDBOX_REMOTE_ON_FAIL\` | fallback (or \`deny\`) |"
  echo "| \`ASH_SANDBOX_REMOTE_LIVE\` | \`1\` enables optional live segment |"
  echo "| \`ASH_SANDBOX_REMOTE_API_KEY\` | required for live; absent → skip pass |"
  echo
  echo "## Raw excerpt"
  echo
  echo '```'
  tail -n 80 "$LOG" || true
  echo '```'
} >"$EVIDENCE"

rm -f "$LOG"
echo "wrote ${EVIDENCE}"
if [[ "$RC" -ne 0 ]]; then
  exit "$RC"
fi
echo "OK remote-sandbox-smoke"
