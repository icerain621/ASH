#!/usr/bin/env bash
# H-09 §7.2 SSE live smoke against running Worker.
set -euo pipefail

BASE="${ASH_WORKER_URL:-}"
RUN_ID="${1:-}"
SPACE="${ASH_SPACE_ID:-local}"
AUTH_HEADER="${ASH_AUTH_HEADER:-}"
TIMEOUT="${ASH_SSE_TIMEOUT_SEC:-8}"

if [[ -z "$BASE" || -z "$RUN_ID" ]]; then
  echo "usage: sse-live-smoke.sh <runId> (requires ASH_WORKER_URL)" >&2
  exit 2
fi

args=(-sS -N --max-time "$TIMEOUT" "${BASE}/api/v1/runs/${RUN_ID}/stream" -H "Accept: text/event-stream" -H "X-ASH-Space-ID: ${SPACE}")
if [[ -n "$AUTH_HEADER" ]]; then
  args+=(-H "Authorization: ${AUTH_HEADER}")
fi

echo "== SSE stream ${RUN_ID} (timeout ${TIMEOUT}s) =="
chunk="$(curl "${args[@]}" | head -c 4096 || true)"
if [[ -z "$chunk" ]]; then
  echo "SSE empty response" >&2
  exit 1
fi
if ! printf '%s' "$chunk" | grep -qE '(^|\n)(data:|event:)'; then
  echo "SSE missing event/data lines:" >&2
  printf '%s\n' "$chunk" | head -20 >&2
  exit 1
fi
printf '%s' "$chunk" | head -c 240
echo ""
echo "OK sse-live-smoke"
