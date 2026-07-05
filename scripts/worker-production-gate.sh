#!/usr/bin/env bash
# H-03 local: Worker with production-like JWT secrets + authenticated live smoke.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
# shellcheck source=_go_env.sh
source "$ROOT/scripts/_go_env.sh"
_ash_go_env_bootstrap "$ROOT"

PORT="${ASH_WORKER_PORT:-18081}"
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
export ASH_AUTH_MODE=jwt
export ASH_JWT_SECRET="${ASH_JWT_SECRET:-prod-jwt-secret-32chars-minimum!!}"
export ASH_SECRET_KEY="${ASH_SECRET_KEY:-prod-data-key-32chars-minimum!!!}"
export ASH_CI_FIXTURE=1
export ASH_WORKER_URL="$BASE"

echo "== start JWT Worker @ ${BASE} =="
go run ./cmd/worker >"$DATA_DIR/worker.log" 2>&1 &
WORKER_PID=$!

deadline=$((SECONDS + 90))
until curl -sf "${BASE}/readyz" >/dev/null 2>&1; do
  if ! kill -0 "$WORKER_PID" 2>/dev/null; then
    tail -40 "$DATA_DIR/worker.log" >&2 || true
    exit 1
  fi
  if (( SECONDS > deadline )); then
    tail -40 "$DATA_DIR/worker.log" >&2 || true
    exit 1
  fi
  sleep 1
done

echo "== dev-login (jwt mode) =="
LOGIN_JSON=$(curl -sf -X POST "${BASE}/api/v1/auth/dev-login")
TOKEN=$(printf '%s' "$LOGIN_JSON" | sed -n 's/.*"token"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -1)
if [[ -z "$TOKEN" ]]; then
  echo "dev-login failed: $LOGIN_JSON" >&2
  exit 1
fi
export ASH_AUTH_HEADER="Bearer ${TOKEN}"

echo "== authenticated /api/v1/runs =="
curl -sf -H "Authorization: ${ASH_AUTH_HEADER}" "${BASE}/api/v1/runs" >/dev/null

echo "== live smoke (JWT) =="
bash scripts/live-smoke.sh

echo "OK worker-production-gate"
