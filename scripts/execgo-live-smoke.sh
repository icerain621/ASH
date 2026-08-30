#!/usr/bin/env bash
# H-06 ExecGo live smoke: execgo-health + Doctor M3-05 (+ optional Worker providers probe).
# Evidence under .ash/evidence/execgo-live-* when ASH_EVIDENCE=1 (default on).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
# shellcheck source=_go_env.sh
source "$ROOT/scripts/_go_env.sh"
_ash_go_env_bootstrap "$ROOT"
# shellcheck source=_evidence.sh
source "$ROOT/scripts/_evidence.sh"

BASE="${ASH_WORKER_URL:-}"
EVIDENCE_ON="${ASH_EVIDENCE:-1}"

if [[ "$EVIDENCE_ON" == "1" ]]; then
  ash_evidence_init execgo-live
  echo "Evidence dir: $ASH_EVIDENCE_DIR"
fi

echo "== H-06 ExecGo live smoke =="

echo "== preflight: execgo-health =="
if [[ "$EVIDENCE_ON" == "1" ]]; then
  ash_evidence_step execgo-health bash scripts/execgo-health.sh
else
  bash scripts/execgo-health.sh
fi

if [[ "${ASH_EXECGO_E2E:-}" != "1" ]]; then
  echo "ASH_EXECGO_E2E must be 1 for live M3-05 (export ASH_EXECGO_E2E=1)" >&2
  exit 1
fi

echo "== Doctor M3 require M3-05 (--agent execgo_codex) =="
if [[ "$EVIDENCE_ON" == "1" ]]; then
  ash_evidence_step doctor-m305 \
    env ASH_EXECGO_E2E=1 go run ./cmd/cli doctor --suite M3 --require M3-05 --agent execgo_codex --format md
else
  ASH_EXECGO_E2E=1 go run ./cmd/cli doctor --suite M3 --require M3-05 --agent execgo_codex --format md
fi

if [[ -n "$BASE" ]]; then
  echo "== providers/agent + readyz @ ${BASE} =="
  PROVIDERS=$(curl -sS "${BASE}/api/v1/providers/agent" || true)
  READYZ=$(curl -sS "${BASE}/readyz" || true)
  if [[ "$EVIDENCE_ON" == "1" ]]; then
    printf '%s\n' "$PROVIDERS" >"$ASH_EVIDENCE_DIR/providers-agent.json"
    printf '%s\n' "$READYZ" >"$ASH_EVIDENCE_DIR/readyz.json"
  fi
  if echo "$PROVIDERS" | grep -qE '"ok"[[:space:]]*:[[:space:]]*true|"execGo"'; then
    echo "OK providers/agent reachable"
    echo "$PROVIDERS" | head -c 400
    echo ""
  else
    echo "WARN: providers/agent unexpected (auth/dev mode?): $PROVIDERS" | head -c 400 >&2
  fi
  if ! echo "$READYZ" | grep -q 'ASH_EXECGO_E2E'; then
    echo "WARN: Worker readyz missing ASH_EXECGO_E2E liveGateHint; start Worker with ASH_EXECGO_E2E=1" >&2
  else
    echo "OK readyz includes ASH_EXECGO_E2E hint"
  fi
else
  echo "== Worker probe skipped (set ASH_WORKER_URL for providers/agent + readyz) =="
fi

echo "OK H-06 ExecGo live smoke"
