#!/usr/bin/env bash
# H-06 ExecGo live smoke: execgo-health + Doctor M3-05 (requires ASH_EXECGO_E2E=1).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
# shellcheck source=_go_env.sh
source "$ROOT/scripts/_go_env.sh"
_ash_go_env_bootstrap "$ROOT"

BASE="${ASH_WORKER_URL:-}"

echo "== H-06 ExecGo live smoke =="

echo "== preflight: execgo-health =="
bash scripts/execgo-health.sh

if [[ "${ASH_EXECGO_E2E:-}" != "1" ]]; then
  echo "ASH_EXECGO_E2E must be 1 for live M3-05 (export ASH_EXECGO_E2E=1)" >&2
  exit 1
fi

echo "== Doctor M3 require M3-05 (--agent execgo_codex) =="
ASH_EXECGO_E2E=1 go run ./cmd/cli doctor --suite M3 --require M3-05 --agent execgo_codex --format md

if [[ -n "$BASE" ]]; then
  echo "== readyz liveGateHints (ASH_EXECGO_E2E) @ ${BASE} =="
  READYZ=$(curl -sS "${BASE}/readyz" || true)
  if ! echo "$READYZ" | grep -q 'ASH_EXECGO_E2E'; then
    echo "WARN: Worker readyz missing ASH_EXECGO_E2E liveGateHint; start Worker with ASH_EXECGO_E2E=1" >&2
    echo "$READYZ" | head -c 400 >&2
  else
    echo "OK readyz includes ASH_EXECGO_E2E hint"
  fi
else
  echo "== readyz hint check skipped (set ASH_WORKER_URL for Worker gate) =="
fi

echo "OK H-06 ExecGo live smoke"
