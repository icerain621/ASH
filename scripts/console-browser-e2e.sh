#!/usr/bin/env bash
# Console browser E2E: ephemeral Worker + Playwright (nav smoke + doctor/readyz + memory + SSE).
# Optional gate (not part of web-gate). Requires Node + Playwright Chromium.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
# shellcheck source=_go_env.sh
source "$ROOT/scripts/_go_env.sh"
_ash_go_env_bootstrap "$ROOT"

PORT="${ASH_WORKER_PORT:-18082}"
PLUGIN_GRPC_PORT="${ASH_PLUGIN_GRPC_PORT:-$((PORT + 1000))}"
BASE="http://127.0.0.1:${PORT}"
DATA_DIR="${TMPDIR:-/tmp}/ash-console-e2e-$$"
mkdir -p "$DATA_DIR"
WORKER_PID=""
SKIP_BUILD="${ASH_CONSOLE_E2E_SKIP_BUILD:-${ASH_SSE_E2E_SKIP_BUILD:-0}}"
SUITE="${ASH_CONSOLE_E2E_SUITE:-all}" # all | smoke | sse

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
export ASH_SCENARIOS_DIR="${ASH_SCENARIOS_DIR:-$ROOT/scenarios}"
# Deterministic agent so POST /runs returns without waiting on ExecGo/Codex.
export ASH_AGENT_EXECUTOR="${ASH_AGENT_EXECUTOR:-static}"

echo "== start ephemeral Worker @ ${BASE} =="
go run ./cmd/worker >"$DATA_DIR/worker.log" 2>&1 &
WORKER_PID=$!

deadline=$((SECONDS + 180))
until curl -sf "${BASE}/readyz" >/dev/null 2>&1; do
  if ! kill -0 "$WORKER_PID" 2>/dev/null; then
    echo "Worker exited early; log:" >&2
    tail -80 "$DATA_DIR/worker.log" >&2 || true
    exit 1
  fi
  if (( SECONDS > deadline )); then
    echo "Worker readyz timeout @ ${BASE}" >&2
    tail -80 "$DATA_DIR/worker.log" >&2 || true
    exit 1
  fi
  sleep 1
done

if ! curl -sf "${BASE}/ui/" | head -c 200 | grep -qiE 'html|ash|root|script'; then
  echo "Worker /ui/ not serving console; check ASH_WEB_DIR / frontend/dist" >&2
  tail -40 "$DATA_DIR/worker.log" >&2 || true
  exit 1
fi

echo "== Playwright console E2E (suite=${SUITE}) =="
(
  cd "$ROOT/frontend"
  case "$SUITE" in
    smoke)
      ASH_WORKER_URL="$BASE" npx playwright test \
        e2e/console-nav-smoke.spec.ts \
        e2e/doctor-readyz.spec.ts \
        e2e/memory-console.spec.ts
      ;;
    sse)
      ASH_WORKER_URL="$BASE" npx playwright test e2e/sse-run-stream.spec.ts
      ;;
    all|*)
      ASH_WORKER_URL="$BASE" npx playwright test
      ;;
  esac
)

echo "OK console-browser-e2e"
