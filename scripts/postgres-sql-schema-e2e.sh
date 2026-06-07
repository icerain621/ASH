#!/usr/bin/env bash
# Postgres schema pilot: ASH_SCHEMA_MODE=sql (golang-migrate only, no GORM AutoMigrate).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
# shellcheck source=_go_env.sh
source "$ROOT/scripts/_go_env.sh"
_ash_go_env_bootstrap "$ROOT"

echo "== start postgres =="
bash scripts/postgres-up.sh
PORT="${ASH_POSTGRES_PORT:-5432}"
export ASH_DATABASE_URL="postgres://ash:ash@127.0.0.1:${PORT}/ash?sslmode=disable"
export ASH_SCHEMA_MODE=sql
export ASH_MIGRATE_E2E=1

echo "== reset postgres schema =="
docker exec ash-postgres-dev psql -U ash -d ash -c \
  'DROP SCHEMA IF EXISTS public CASCADE; CREATE SCHEMA public; GRANT ALL ON SCHEMA public TO ash; GRANT ALL ON SCHEMA public TO public;'

echo "== sql-only migrate via store.Open =="
E2E_DIR="${ASH_E2E_DATA_DIR:-$(mktemp -d)}"
export ASH_DATA_DIR="$E2E_DIR"
go test -tags=integration ./internal/store/ -run TestPostgresSQLSchemaModeE2E -count=1

echo "== doctor M3 (incl. M3-08 sql schema) =="
go run ./cmd/cli doctor --suite M3

echo "OK postgres sql-schema e2e (mode=sql, data dir: $E2E_DIR)"
