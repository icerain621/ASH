#!/usr/bin/env bash
# ACP contract smoke (Sprint DX4): unit tests + mock control-plane Probe/Execute.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
# shellcheck source=_go_env.sh
source "$ROOT/scripts/_go_env.sh"
_ash_go_env_bootstrap "$ROOT"

echo "== agentexec ACP contract unit tests =="
go test ./internal/agentexec/ -count=1 -run 'TestACP|TestProbeACP|TestResolve'

MOCK_LOG="$(mktemp "${TMPDIR:-/tmp}/ash-acp-mock.XXXXXX.log")"
cleanup() {
  if [[ -n "${MOCK_PID:-}" ]] && kill -0 "$MOCK_PID" 2>/dev/null; then
    kill "$MOCK_PID" 2>/dev/null || true
    wait "$MOCK_PID" 2>/dev/null || true
  fi
  rm -f "$MOCK_LOG"
}
trap cleanup EXIT

echo "== start acp-mock =="
# First line of stdout is the base URL; logs go to stderr.
go run ./cmd/acp-mock -addr 127.0.0.1:0 >"$MOCK_LOG" 2>"${MOCK_LOG}.err" &
MOCK_PID=$!

BASE=""
for _ in $(seq 1 80); do
  if [[ -s "$MOCK_LOG" ]]; then
    BASE="$(head -n 1 "$MOCK_LOG" | tr -d '\r\n')"
    if [[ "$BASE" == http://* ]]; then
      break
    fi
  fi
  if ! kill -0 "$MOCK_PID" 2>/dev/null; then
    echo "acp-mock exited early" >&2
    cat "${MOCK_LOG}.err" >&2 || true
    cat "$MOCK_LOG" >&2 || true
    exit 1
  fi
  sleep 0.1
done
if [[ -z "$BASE" || "$BASE" != http://* ]]; then
  echo "acp-mock failed to publish base URL" >&2
  cat "${MOCK_LOG}.err" >&2 || true
  cat "$MOCK_LOG" >&2 || true
  exit 1
fi
echo "mock=$BASE"

export ASH_ACP_ENDPOINT="$BASE"
export ASH_ACP_URL="$BASE"

echo "== ProbeACP + Execute against mock (env) =="
go test ./internal/agentexec/ -count=1 -run 'TestACPSmokeAgainstEnv' -v

echo "OK acp-smoke"
