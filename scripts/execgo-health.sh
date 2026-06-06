#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
LOCAL_CLI="$ROOT_DIR/.ash/execgo/execgo/bin/execgocli"
CLI="${EXECGO_EXECGOCLI:-execgocli}"
CODEX_BIN="${ASH_CODEX_BIN:-codex}"

fail() {
  local class="$1"
  local message="$2"
  echo "ExecGo health failed: ${class}" >&2
  echo "$message" >&2
  exit 1
}

if ! command -v "$CLI" >/dev/null 2>&1; then
  if [[ -x "$LOCAL_CLI" ]]; then
    CLI="$LOCAL_CLI"
  else
    fail "execgocli missing" "Run: make execgo-bootstrap"
  fi
fi

if ! command -v "$CODEX_BIN" >/dev/null 2>&1; then
  fail "codex cli missing" "Set ASH_CODEX_BIN to the Codex executable used by ASH."
fi

export EXECGO_URL="${EXECGO_URL:-http://127.0.0.1:8080}"
export EXECGO_RUNTIME_URL="${EXECGO_RUNTIME_URL:-http://127.0.0.1:18080}"

echo "ExecGo URL: $EXECGO_URL"
echo "ExecGo Runtime URL: $EXECGO_RUNTIME_URL"
echo "execgocli: $CLI"
echo "Codex CLI: $CODEX_BIN"

run_json_check() {
  local label="$1"
  shift
  local output
  local status
  set +e
  output="$("$CLI" "$@" 2>&1)"
  status=$?
  set -e
  if [[ $status -ne 0 ]]; then
    case "$label" in
      health)
        fail "control plane unreachable" "$output"
        ;;
      tools)
        fail "runtime/tools unavailable" "$output"
        ;;
      *)
        fail "$label failed" "$output"
        ;;
    esac
  fi
  if ! printf '%s' "$output" | grep -q '"ok"'; then
    fail "execgocli JSON parse failed" "$output"
  fi
  echo "$output"
}

run_json_check "health" health
run_json_check "tools" tools

echo "ExecGo bridge health check passed."
