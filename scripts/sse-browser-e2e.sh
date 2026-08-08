#!/usr/bin/env bash
# P2-4 SSE browser E2E: ephemeral Worker + Playwright Chromium against /ui/runs.
# Optional gate (not part of web-gate). Requires Node + Playwright browsers.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
# shellcheck source=_go_env.sh
source "$ROOT/scripts/_go_env.sh"
_ash_go_env_bootstrap "$ROOT"

PORT="${ASH_WORKER_PORT:-18082}"
# Derive a free-ish gRPC port from HTTP port so parallel/local leftovers don't collide.
PLUGIN_GRPC_PORT="${ASH_PLUGIN_GRPC_PORT:-$((PORT + 1000))}"
BASE="http://127.0.0.1:${PORT}"
DATA_DIR="$(mktemp -d 2>/dev/null || mktemp -d -t ash-sse-e2e)"
WORKER_PID=""
SKIP_BUILD="${ASH_SSE_E2E_SKIP_BUILD:-0}"

cleanup() {
  if [[ -n "${WORKER_PID}" ]] && kill -0 "$WORKER_PID" 2>/dev/null; then
    kill "$WORKER_PID" 2>/dev/null || true
    wait "$WORKER_PID" 2>/dev/null || true
  fi
  rm -rf "$DATA_DIR"
}
trap cleanup EXIT

export NPM_CONFIG_AUDIT=false

echo "== frontend deps + Playwright Chromium =="
(
  cd "$ROOT/frontend"
  if [[ -x node_modules/.bin/playwright ]]; then
    echo "reuse node_modules"
  elif [[ -f package-lock.json ]]; then
    npm ci --no-audit --no-fund || npm install --no-audit --no-fund
  else
    npm install --no-audit --no-fund
  fi
  npx playwright install chromium
)

if [[ "$SKIP_BUILD" != "1" ]]; then
  echo "== web-build (Worker serves frontend/dist) =="
  (
    cd "$ROOT/frontend"
    npm run build
  )
fi

if [[ ! -f "$ROOT/frontend/dist/index.html" ]]; then
  echo "missing frontend/dist/index.html — run make web-build" >&2
  exit 1
fi

export ASH_DATA_DIR="$DATA_DIR"
export ASH_HTTP_ADDR=":${PORT}"
export ASH_AUTH_MODE="${ASH_AUTH_MODE:-dev}"
export ASH_CI_FIXTURE=1
export ASH_WORKER_URL="$BASE"
export ASH_WEB_DIR="${ASH_WEB_DIR:-$ROOT/frontend/dist}"
export ASH_PLUGIN_GRPC_ADDR="${ASH_PLUGIN_GRPC_ADDR:-127.0.0.1:${PLUGIN_GRPC_PORT}}"

echo "== start ephemeral Worker @ ${BASE} =="
go run ./cmd/worker >"$DATA_DIR/worker.log" 2>&1 &
WORKER_PID=$!

deadline=$((SECONDS + 90))
until curl -sf "${BASE}/readyz" >/dev/null 2>&1; do
  if ! kill -0 "$WORKER_PID" 2>/dev/null; then
    echo "Worker exited early; log:" >&2
    tail -40 "$DATA_DIR/worker.log" >&2 || true
    exit 1
  fi
  if (( SECONDS > deadline )); then
    echo "Worker readyz timeout @ ${BASE}" >&2
    tail -40 "$DATA_DIR/worker.log" >&2 || true
    exit 1
  fi
  sleep 1
done

if ! curl -sf "${BASE}/ui/" | head -c 200 | grep -qiE 'html|ash|root|script'; then
  echo "Worker /ui/ not serving console; check ASH_WEB_DIR / frontend/dist" >&2
  tail -40 "$DATA_DIR/worker.log" >&2 || true
  exit 1
fi

echo "== Playwright SSE browser smoke =="
(
  cd "$ROOT/frontend"
  ASH_WORKER_URL="$BASE" npx playwright test e2e/sse-run-stream.spec.ts
)

echo "OK sse-browser-e2e"
