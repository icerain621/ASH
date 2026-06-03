#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
LOCAL_CLI="$ROOT_DIR/.ash/execgo/execgo/bin/execgocli"
CLI="${EXECGO_EXECGOCLI:-execgocli}"
CODEX_BIN="${ASH_CODEX_BIN:-codex}"

if ! command -v "$CLI" >/dev/null 2>&1; then
  if [[ -x "$LOCAL_CLI" ]]; then
    CLI="$LOCAL_CLI"
  else
    echo "execgocli is unavailable." >&2
    echo "Run: make execgo-bootstrap" >&2
    exit 1
  fi
fi

if ! command -v "$CODEX_BIN" >/dev/null 2>&1; then
  echo "Codex CLI is unavailable: $CODEX_BIN" >&2
  echo "Set ASH_CODEX_BIN to the Codex executable used by ASH." >&2
  exit 1
fi

export EXECGO_URL="${EXECGO_URL:-http://127.0.0.1:8080}"
export EXECGO_RUNTIME_URL="${EXECGO_RUNTIME_URL:-http://127.0.0.1:18080}"

echo "ExecGo URL: $EXECGO_URL"
echo "ExecGo Runtime URL: $EXECGO_RUNTIME_URL"
echo "execgocli: $CLI"
echo "Codex CLI: $CODEX_BIN"

"$CLI" health
"$CLI" tools

echo "ExecGo bridge health check passed."
