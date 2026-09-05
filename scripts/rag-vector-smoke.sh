#!/usr/bin/env bash
# RAG Vector smoke (Sprint DX17): Index embed + Query vector lane (mocked store, no live Qdrant).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
# shellcheck source=_go_env.sh
source "$ROOT/scripts/_go_env.sh"
_ash_go_env_bootstrap "$ROOT"

echo "== rag vector package tests =="
go test ./internal/rag/ -count=1 -run 'TestQueryVectorLane|TestIndexEmbed|TestQuerySucceedsWithoutVector|TestProfileReportsVector'

echo "OK rag-vector-smoke"
