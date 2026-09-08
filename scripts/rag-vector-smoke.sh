#!/usr/bin/env bash
# RAG Vector smoke (DX17/DX25/DX26 + DX43–DX47): mock/backends/prefer; optional live embed.
# No live Chroma/Milvus/Qdrant required — httptest + mock cover adapters.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
# shellcheck source=_go_env.sh
source "$ROOT/scripts/_go_env.sh"
_ash_go_env_bootstrap "$ROOT"

EVIDENCE="$ROOT/doc/evidence/rag-vector-smoke-latest.md"
STAMP="$(date -u +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || date -u)"
OS_NAME="$(uname -s 2>/dev/null || echo unknown)"
LOG="$(mktemp 2>/dev/null || mktemp -t ash-rag-vec)"

RUN_FILTER='TestQueryVectorLane|TestQueryPrefer|TestEffectivePrefer|TestIndexEmbed|TestQuerySucceedsWithoutVector|TestProfileReportsVector|TestProfileReportsHybridVector|TestQdrantClient|TestOpenAICompat|TestResolveEmbedder|TestEmbeddingsEndpoint|TestEmbedderKind|TestVectorConfig|TestResolveVector|TestResolveChroma|TestResolveMilvus|TestProbeVector|TestChromaClient|TestMilvusClient'

echo "== rag vector package tests (DX25–DX47) =="
set +e
go test ./internal/rag/ -count=1 -timeout 180s -run "$RUN_FILTER" 2>&1 | tee "$LOG"
RC=${PIPESTATUS[0]}
set -e

STATUS="pass"
if [[ "$RC" -ne 0 ]]; then
  STATUS="fail"
fi

LIVE_STATUS="skipped"
LIVE_NOTE="set ASH_EMBED_LIVE=1 + ASH_EMBED_BASE_URL for live OpenAI-compat embed"
if [[ "${ASH_EMBED_LIVE:-}" == "1" ]]; then
  if [[ -z "${ASH_EMBED_BASE_URL:-}" ]]; then
    echo "ASH_EMBED_LIVE=1 requires ASH_EMBED_BASE_URL" >&2
    STATUS="fail"
    RC=1
    LIVE_STATUS="fail"
    LIVE_NOTE="ASH_EMBED_LIVE=1 but ASH_EMBED_BASE_URL unset"
  else
    echo "== live OpenAI-compat embed (ASH_EMBED_LIVE=1) =="
    LIVE_LOG="$(mktemp 2>/dev/null || mktemp -t ash-embed-live)"
    set +e
    go test ./internal/rag/ -count=1 -run TestOpenAICompatEmbedderLive -tags=liveembed 2>&1 | tee "$LIVE_LOG"
    LIVE_RC=${PIPESTATUS[0]}
    set -e
    if [[ "$LIVE_RC" -eq 0 ]]; then
      LIVE_STATUS="pass"
      LIVE_NOTE="TestOpenAICompatEmbedderLive ok"
    else
      LIVE_STATUS="fail"
      LIVE_NOTE="TestOpenAICompatEmbedderLive exit ${LIVE_RC}"
      STATUS="fail"
      RC="$LIVE_RC"
    fi
    {
      echo
      echo "## Live embed excerpt"
      echo
      echo '```'
      tail -n 40 "$LIVE_LOG" || true
      echo '```'
    } >>"$LOG"
    rm -f "$LIVE_LOG"
  fi
fi

{
  echo "# rag-vector-smoke"
  echo
  echo "- stamp: \`${STAMP}\`"
  echo "- os: \`${OS_NAME}\`"
  echo "- status: **${STATUS}**"
  echo "- live embed: **${LIVE_STATUS}** (${LIVE_NOTE})"
  echo "- filter: \`${RUN_FILTER}\`"
  echo
  echo "## Output"
  echo
  echo '```'
  tail -n 80 "$LOG" || true
  echo '```'
} >"$EVIDENCE"
rm -f "$LOG"

echo "Wrote $EVIDENCE"
if [[ "$STATUS" != "pass" ]]; then
  echo "FAIL rag-vector-smoke" >&2
  exit "$RC"
fi
echo "OK rag-vector-smoke"
