#!/usr/bin/env bash
# End-to-end sqlite → postgres migration (requires Docker).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

KEEP_DATA=0
KEEP_POSTGRES=0
for arg in "$@"; do
  case "$arg" in
    --keep-data) KEEP_DATA=1 ;;
    --keep-postgres) KEEP_POSTGRES=1 ;;
    *)
      echo "Usage: $0 [--keep-data] [--keep-postgres]" >&2
      exit 2
      ;;
  esac
done

E2E_DIR="${ASH_E2E_DATA_DIR:-$(mktemp -d)}"
WORKER_PORT="${ASH_E2E_WORKER_PORT:-18081}"
WORKER_PID=""
cleanup() {
  if [[ -n "$WORKER_PID" ]]; then
    kill "$WORKER_PID" >/dev/null 2>&1 || true
    wait "$WORKER_PID" >/dev/null 2>&1 || true
  fi
  if [[ "$KEEP_POSTGRES" -eq 0 ]]; then
    bash scripts/postgres-down.sh >/dev/null 2>&1 || true
  fi
  if [[ "$KEEP_DATA" -eq 0 && -d "$E2E_DIR" ]]; then
    rm -rf "$E2E_DIR"
  fi
}
trap cleanup EXIT

if ! command -v curl >/dev/null 2>&1; then
  echo "curl is required for readyz polling" >&2
  exit 2
fi

choose_port() {
  if command -v python3 >/dev/null 2>&1; then
    python3 - <<'PY'
import socket
with socket.socket() as s:
    s.bind(("127.0.0.1", 0))
    print(s.getsockname()[1])
PY
    return
  fi
  echo "15432"
}

if [[ -z "${ASH_POSTGRES_PORT:-}" ]]; then
  export ASH_POSTGRES_PORT="$(choose_port)"
fi

echo "== reset postgres =="
bash scripts/postgres-down.sh >/dev/null 2>&1 || true

echo "== start postgres =="
bash scripts/postgres-up.sh
PORT="${ASH_POSTGRES_PORT:-5432}"
POSTGRES_URL="postgres://ash:ash@127.0.0.1:${PORT}/ash?sslmode=disable"

export ASH_DATA_DIR="$E2E_DIR"
SQLITE_PATH="${ASH_SQLITE_PATH:-$E2E_DIR/ash.db}"

echo "== seed sqlite at $E2E_DIR =="
env -u ASH_DATABASE_URL -u ASH_MIGRATE_POSTGRES_URL \
  ASH_DATA_DIR="$ASH_DATA_DIR" \
  go run ./cmd/cli doctor --suite M3 --format md --agent static

echo "== migrate plan =="
go run ./cmd/cli migrate plan \
  --data-dir "$ASH_DATA_DIR" \
  --sqlite "$SQLITE_PATH" \
  --postgres "$POSTGRES_URL"

echo "== migrate copy =="
go run ./cmd/cli migrate copy \
  --data-dir "$ASH_DATA_DIR" \
  --sqlite "$SQLITE_PATH" \
  --postgres "$POSTGRES_URL"

echo "== migrate verify =="
go run ./cmd/cli migrate verify \
  --data-dir "$ASH_DATA_DIR" \
  --sqlite "$SQLITE_PATH" \
  --postgres "$POSTGRES_URL"

echo "== doctor M3 (incl. M3-04 when ASH_MIGRATE_E2E=1) =="
env -u ASH_DATABASE_URL \
  ASH_DATA_DIR="$ASH_DATA_DIR" \
  ASH_MIGRATE_E2E=1 \
  ASH_MIGRATE_POSTGRES_URL="$POSTGRES_URL" \
  go run ./cmd/cli doctor --suite M3 --format md --agent static

echo "== worker readyz with postgres primary =="
WORKER_DATA_DIR="$E2E_DIR/worker-postgres"
WORKER_LOG="$E2E_DIR/worker-postgres.log"
ASH_DATA_DIR="$WORKER_DATA_DIR" \
  ASH_DATABASE_URL="$POSTGRES_URL" \
  ASH_HTTP_ADDR="127.0.0.1:${WORKER_PORT}" \
  go run ./cmd/worker >"$WORKER_LOG" 2>&1 &
WORKER_PID=$!

READY_URL="http://127.0.0.1:${WORKER_PORT}/readyz"
for _ in $(seq 1 60); do
  if curl -fsS "$READY_URL" >/dev/null 2>&1; then
    echo "readyz OK: $READY_URL"
    break
  fi
  sleep 1
done
if ! curl -fsS "$READY_URL" >/dev/null 2>&1; then
  echo "worker readyz failed; log follows:" >&2
  cat "$WORKER_LOG" >&2
  exit 1
fi

echo "== doctor ALL regression (static agent) =="
env -u ASH_DATABASE_URL -u ASH_MIGRATE_POSTGRES_URL \
  ASH_DATA_DIR="$E2E_DIR/doctor-all" \
  go run ./cmd/cli doctor --suite ALL --format md --agent static

echo "OK postgres e2e migrate (data dir: $E2E_DIR)"
