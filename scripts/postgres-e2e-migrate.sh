#!/usr/bin/env bash
# End-to-end sqlite → postgres migration (requires Docker).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

KEEP_DATA=0
if [[ "${1:-}" == "--keep-data" ]]; then
  KEEP_DATA=1
fi

E2E_DIR="${ASH_E2E_DATA_DIR:-$(mktemp -d)}"
cleanup() {
  if [[ "$KEEP_DATA" -eq 0 && -d "$E2E_DIR" ]]; then
    rm -rf "$E2E_DIR"
  fi
}
trap cleanup EXIT

echo "== start postgres =="
bash scripts/postgres-up.sh
PORT="${ASH_POSTGRES_PORT:-5432}"
export ASH_DATABASE_URL="postgres://ash:ash@127.0.0.1:${PORT}/ash?sslmode=disable"

export ASH_DATA_DIR="$E2E_DIR"
export ASH_MIGRATE_E2E=1
SQLITE_PATH="${ASH_SQLITE_PATH:-$E2E_DIR/ash.db}"

echo "== seed sqlite at $E2E_DIR =="
go run ./cmd/cli doctor --suite M3

echo "== migrate plan =="
go run ./cmd/cli migrate plan \
  --data-dir "$ASH_DATA_DIR" \
  --sqlite "$SQLITE_PATH" \
  --postgres "$ASH_DATABASE_URL"

echo "== migrate copy =="
go run ./cmd/cli migrate copy \
  --data-dir "$ASH_DATA_DIR" \
  --sqlite "$SQLITE_PATH" \
  --postgres "$ASH_DATABASE_URL"

echo "== migrate verify =="
go run ./cmd/cli migrate verify \
  --data-dir "$ASH_DATA_DIR" \
  --sqlite "$SQLITE_PATH" \
  --postgres "$ASH_DATABASE_URL"

echo "== doctor M3 (incl. M3-04 when ASH_MIGRATE_E2E=1) =="
go run ./cmd/cli doctor --suite M3

echo "OK postgres e2e migrate (data dir: $E2E_DIR)"
