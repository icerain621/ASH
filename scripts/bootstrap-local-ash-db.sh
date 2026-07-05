#!/usr/bin/env bash
# Ensure .ash/ash.db exists for backup / pre-migrate gates (MVP §6 local drill).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
# shellcheck source=_go_env.sh
source "$ROOT/scripts/_go_env.sh"
_ash_go_env_bootstrap "$ROOT"

DATA_DIR="${ASH_DATA_DIR:-$ROOT/.ash}"
SQLITE="${ASH_SQLITE_PATH:-$DATA_DIR/ash.db}"

if [[ -f "$SQLITE" ]]; then
  echo "OK existing sqlite: $SQLITE"
  exit 0
fi

PORT="${ASH_BOOTSTRAP_PORT:-18079}"
WORKER_PID=""

cleanup() {
  if [[ -n "$WORKER_PID" ]] && kill -0 "$WORKER_PID" 2>/dev/null; then
    kill "$WORKER_PID" 2>/dev/null || true
    wait "$WORKER_PID" 2>/dev/null || true
  fi
}
trap cleanup EXIT

mkdir -p "$DATA_DIR"
export ASH_DATA_DIR="$DATA_DIR"
export ASH_SQLITE_PATH="$SQLITE"
export ASH_HTTP_ADDR=":${PORT}"
export ASH_AUTH_MODE="${ASH_AUTH_MODE:-dev}"

echo "== bootstrap sqlite @ ${SQLITE} =="
go run ./cmd/worker >"$DATA_DIR/bootstrap.log" 2>&1 &
WORKER_PID=$!

deadline=$((SECONDS + 90))
until curl -sf "http://127.0.0.1:${PORT}/readyz" >/dev/null 2>&1; do
  if ! kill -0 "$WORKER_PID" 2>/dev/null; then
    tail -40 "$DATA_DIR/bootstrap.log" >&2 || true
    exit 1
  fi
  if (( SECONDS > deadline )); then
    tail -40 "$DATA_DIR/bootstrap.log" >&2 || true
    exit 1
  fi
  sleep 1
done

if [[ ! -f "$SQLITE" ]]; then
  echo "Worker ready but sqlite missing: $SQLITE" >&2
  exit 1
fi
echo "OK bootstrap-local-ash-db: $SQLITE"
