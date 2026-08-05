#!/usr/bin/env bash
# End-to-end sqlite → postgres migration (requires Docker).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
# shellcheck source=_go_env.sh
source "$ROOT/scripts/_go_env.sh"
_ash_go_env_bootstrap "$ROOT"

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
# shellcheck source=postgres-up.sh
source scripts/postgres-up.sh
PORT="${ASH_POSTGRES_PORT:-5432}"
export ASH_DATABASE_URL="postgres://ash:ash@127.0.0.1:${PORT}/ash?sslmode=disable"

export ASH_DATA_DIR="$E2E_DIR"
SQLITE_PATH="${ASH_SQLITE_PATH:-$E2E_DIR/ash.db}"

echo "== reset postgres schema =="
docker exec ash-postgres-dev psql -U ash -d ash -c \
  'DROP SCHEMA IF EXISTS public CASCADE; CREATE SCHEMA public; GRANT ALL ON SCHEMA public TO ash; GRANT ALL ON SCHEMA public TO public;'

echo "== seed sqlite at $E2E_DIR (sqlite-only) =="
env -u ASH_DATABASE_URL -u ASH_MIGRATE_E2E go run ./cmd/cli doctor --suite M3

export ASH_MIGRATE_E2E=1

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

echo "== ensure postgres app roles =="
bash scripts/postgres-ensure-app-role.sh

echo "== migrate verify =="
go run ./cmd/cli migrate verify \
  --data-dir "$ASH_DATA_DIR" \
  --sqlite "$SQLITE_PATH" \
  --postgres "$ASH_DATABASE_URL"

echo "== doctor M3 (incl. M3-04 when ASH_MIGRATE_E2E=1) =="
go run ./cmd/cli doctor --suite M3 --agent static

echo "== assert M3-04 live migrate verify =="
bash scripts/postgres-doctor-assert.sh M3 M3-04

echo "== worker readyz on postgres =="
go test -tags=integration ./internal/api/ -run TestPostgresReadyzProbe -count=1

echo "== integration test =="
go test -tags=integration ./internal/store/ -run TestMigratorSQLiteToPostgresE2E -count=1

echo "== postgres RLS + ash_app smoke =="
export ASH_DATABASE_APP_URL="postgres://ash_app:ash_app@127.0.0.1:${PORT}/ash?sslmode=disable"
export ASH_POSTGRES_RLS=1
export ASH_POSTGRES_RLS_FORCE=1
bash scripts/postgres-rls-e2e.sh

echo "== readyz + scale ops parity on postgres =="
go test -tags=integration ./internal/api/ -run TestPostgresReadyzWithRLS -count=1
go test -tags=integration ./internal/api/ -run TestPostgresReadyzScaleParity -count=1

echo "== doctor M3 with RLS env =="
# Unset ASH_MIGRATE_E2E: M3-04 was already asserted above. Re-running verify after RLS/doctor
# seeding drifts postgres row counts vs the original sqlite snapshot (e.g. users 0→2).
env -u ASH_MIGRATE_E2E \
  ASH_POSTGRES_RLS=1 ASH_POSTGRES_RLS_FORCE=1 ASH_DATABASE_APP_URL="$ASH_DATABASE_APP_URL" \
  go run ./cmd/cli doctor --suite M3 --agent static

echo "== assert M3-06/07 live RLS + ash_app =="
bash scripts/postgres-doctor-assert.sh M3 M3-06,M3-07

echo "== doctor TR3 on postgres (TR3-06 fts; TR3-10 readyz contract) =="
env -u ASH_MIGRATE_E2E go run ./cmd/cli doctor --suite TR3 --agent static

bash scripts/postgres-doctor-assert.sh TR3 TR3-06,TR3-10

echo "OK postgres e2e migrate (data dir: $E2E_DIR)"
