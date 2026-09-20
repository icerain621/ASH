#!/usr/bin/env bash
# Session RPC smoke (Sprint VX31): ServeRPC idle session.start + optional CLI pipe.
# No new tables. Does not tag.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
# shellcheck source=_go_env.sh
source "$ROOT/scripts/_go_env.sh"
_ash_go_env_bootstrap "$ROOT"

EVIDENCE="$ROOT/doc/evidence/session-rpc-smoke-latest.md"
STAMP="$(date -u +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || date -u)"
OS_NAME="$(uname -s 2>/dev/null || echo unknown)"
LOG="$(mktemp 2>/dev/null || mktemp -t ash-session-rpc)"

echo "== session ServeRPC unit (idle session.start) =="
set +e
go test ./internal/session/ -count=1 -timeout 120s -run 'TestServeRPCSessionStartIdle' 2>&1 | tee "$LOG"
RC=${PIPESTATUS[0]}
set -e

STATUS="pass"
if [[ "$RC" -ne 0 ]]; then
  STATUS="fail"
fi

CLI_STATUS="skipped"
CLI_NOTE="set ASH_SESSION_RPC_CLI=1 to pipe session.start through go run ./cmd/cli"
if [[ "${ASH_SESSION_RPC_CLI:-}" == "1" ]] && [[ "$STATUS" == "pass" ]]; then
  echo "== CLI ash session rpc (temp data dir) =="
  TMPDATA="$(mktemp -d 2>/dev/null || mktemp -d -t ash-rpc-data)"
  CLI_LOG="$(mktemp 2>/dev/null || mktemp -t ash-rpc-cli)"
  set +e
  printf '%s\n' '{"type":"session.start","repoRoot":"."}' | \
    env ASH_DATA_DIR="$TMPDATA" go run ./cmd/cli session rpc --agent static >"$CLI_LOG" 2>"${CLI_LOG}.err"
  CLI_RC=$?
  set -e
  if [[ "$CLI_RC" -eq 0 ]] && grep -q 'session.started' "$CLI_LOG"; then
    CLI_STATUS="pass"
    CLI_NOTE="CLI session.started ok"
  else
    CLI_STATUS="fail"
    CLI_NOTE="CLI exit ${CLI_RC}"
    STATUS="fail"
    RC="$CLI_RC"
    {
      echo
      echo "## CLI stderr"
      echo
      echo '```'
      tail -n 40 "${CLI_LOG}.err" || true
      echo '```'
    } >>"$LOG"
  fi
  rm -rf "$TMPDATA"
  rm -f "$CLI_LOG" "${CLI_LOG}.err"
fi

{
  echo "# Session RPC smoke evidence (VX31)"
  echo
  echo "| Field | Value |"
  echo "|-------|--------|"
  echo "| Status | **${STATUS}** (exit ${RC}) |"
  echo "| Platform | ${OS_NAME} |"
  echo "| Date | ${STAMP} |"
  echo "| Unit | TestServeRPCSessionStartIdle |"
  echo "| CLI pipe | **${CLI_STATUS}** — ${CLI_NOTE} |"
  echo
  echo "## Output excerpt"
  echo
  echo '```'
  tail -n 30 "$LOG" || true
  echo '```'
} >"$EVIDENCE"

rm -f "$LOG"
echo "wrote $EVIDENCE"
if [[ "$STATUS" != "pass" ]]; then
  exit "$RC"
fi
echo "OK session-rpc-smoke"
