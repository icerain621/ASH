#!/usr/bin/env bash
# H-03 local Worker + H-04~H-09 live smoke (ephemeral Worker process).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
# shellcheck source=_go_env.sh
source "$ROOT/scripts/_go_env.sh"
_ash_go_env_bootstrap "$ROOT"

PORT="${ASH_WORKER_PORT:-18080}"
BASE="http://127.0.0.1:${PORT}"
DATA_DIR="$(mktemp -d)"
WORKER_PID=""

cleanup() {
  if [[ -n "$WORKER_PID" ]] && kill -0 "$WORKER_PID" 2>/dev/null; then
    kill "$WORKER_PID" 2>/dev/null || true
    wait "$WORKER_PID" 2>/dev/null || true
  fi
  rm -rf "$DATA_DIR"
}
trap cleanup EXIT

export ASH_DATA_DIR="$DATA_DIR"
export ASH_HTTP_ADDR=":${PORT}"
export ASH_AUTH_MODE="${ASH_AUTH_MODE:-dev}"
export ASH_CI_FIXTURE=1
export ASH_WORKER_URL="$BASE"

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

echo "== readyz ok =="
curl -sf "${BASE}/readyz" | head -c 500
echo ""

echo "== live smoke against ephemeral Worker =="
bash scripts/live-smoke.sh

echo "OK worker-local-gate"
