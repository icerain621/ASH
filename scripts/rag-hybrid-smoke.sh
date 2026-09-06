#!/usr/bin/env bash
# RAG Hybrid smoke (Sprint DX9/DX16/DX20): RebuildSymbols + Hybrid Query + fallback.
# symbolSource (regex|ctags|treesitter) asserted in package tests.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
# shellcheck source=_go_env.sh
source "$ROOT/scripts/_go_env.sh"
_ash_go_env_bootstrap "$ROOT"

echo "== rag hybrid package tests =="
go test ./internal/rag/ -count=1 -run 'TestRebuildSymbolsUpserts|TestRebuildSymbolsFallsBack|TestRebuildSymbolsTreesitter|TestHybridQuery|TestQueryFallsBack|TestTreeSitter|TestResolveSymbolIndexer'

echo "OK rag-hybrid-smoke"
