#!/usr/bin/env bash
# RAG LSP smoke (Sprint DX31–DX35): session reuse, hover/def/refs, expandRefs, harden envs.
# Uses fake-gopls; no live language server required.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
# shellcheck source=_go_env.sh
source "$ROOT/scripts/_go_env.sh"
_ash_go_env_bootstrap "$ROOT"

EVIDENCE="$ROOT/doc/evidence/rag-lsp-smoke-latest.md"
STAMP="$(date -u +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || date -u)"
OS_NAME="$(uname -s 2>/dev/null || echo unknown)"
LOG="$(mktemp 2>/dev/null || mktemp -t ash-rag-lsp)"

echo "== rag lsp package tests =="
set +e
go test ./internal/rag/ -count=1 -timeout 180s -run 'TestLSP|TestHover|TestReferences|TestQueryExpandRefs|TestParseHover|TestRebuildSymbolsLSP|TestResolveSymbolIndexerForcedLSP|TestResolveSymbolIndexerLSPViaEnvFlag' 2>&1 | tee "$LOG"
RC=${PIPESTATUS[0]}
set -e

STATUS="pass"
if [[ "$RC" -ne 0 ]]; then
  STATUS="fail"
fi

{
  echo "# RAG LSP smoke evidence (DX35)"
  echo
  echo "| Field | Value |"
  echo "|-------|--------|"
  echo "| Status | **${STATUS}** (exit ${RC}) |"
  echo "| Platform | ${OS_NAME} |"
  echo "| Date | ${STAMP} |"
  echo "| Scope | session / hover / definition / references / expandRefs / harden |"
  echo "| Live LS | not required (fake-gopls) |"
  echo
  echo "## Env (harden)"
  echo
  echo "| Variable | Default |"
  echo "|----------|---------|"
  echo "| \`ASH_RAG_LSP_SESSION\` | on (\`0\` = one-shot) |"
  echo "| \`ASH_RAG_LSP_IDLE_SEC\` | 30 |"
  echo "| \`ASH_RAG_LSP_TIMEOUT_SEC\` | 20 (max 300) |"
  echo "| \`ASH_RAG_LSP_MAX_OPEN_DOCS\` | 64 |"
  echo
  echo "## Raw excerpt"
  echo
  echo '```'
  tail -n 60 "$LOG" || true
  echo '```'
} >"$EVIDENCE"

rm -f "$LOG"
echo "wrote ${EVIDENCE}"
if [[ "$RC" -ne 0 ]]; then
  exit "$RC"
fi
echo "OK rag-lsp-smoke"
