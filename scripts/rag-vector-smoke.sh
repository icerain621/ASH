#!/usr/bin/env bash
# RAG Vector smoke (DX17/DX25/DX26): mock store + embedder unit tests; optional live embed.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
# shellcheck source=_go_env.sh
source "$ROOT/scripts/_go_env.sh"
_ash_go_env_bootstrap "$ROOT"

echo "== rag vector package tests (DX25/DX26) =="
go test ./internal/rag/ -count=1 -run 'TestQueryVectorLane|TestQueryPreferVector|TestIndexEmbed|TestQuerySucceedsWithoutVector|TestProfileReportsVector|TestProfileReportsHybridVector|TestQdrantClientCreatesCollection|TestQdrantClientUpsertAndSearch|TestOpenAICompat|TestResolveEmbedder|TestEmbeddingsEndpoint|TestEmbedderKind'

if [[ "${ASH_EMBED_LIVE:-}" == "1" ]]; then
  if [[ -z "${ASH_EMBED_BASE_URL:-}" ]]; then
    echo "ASH_EMBED_LIVE=1 requires ASH_EMBED_BASE_URL" >&2
    exit 1
  fi
  echo "== live OpenAI-compat embed (ASH_EMBED_LIVE=1) =="
  go test ./internal/rag/ -count=1 -run TestOpenAICompatEmbedderLive -tags=liveembed
fi

echo "OK rag-vector-smoke"
