#!/usr/bin/env bash
# Postgres schema pilot: ASH_SCHEMA_MODE=sql (golang-migrate only, no GORM AutoMigrate).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
# shellcheck source=_go_env.sh
source "$ROOT/scripts/_go_env.sh"
_ash_go_env_bootstrap "$ROOT"

echo "== start postgres =="
# shellcheck source=postgres-up.sh
source scripts/postgres-up.sh
export ASH_DATABASE_URL="${ASH_DATABASE_URL}"
export ASH_SCHEMA_MODE=sql
export ASH_MIGRATE_E2E=1

echo "== reset postgres schema =="
docker exec ash-postgres-dev psql -U ash -d ash -c \
  'DROP SCHEMA IF EXISTS public CASCADE; CREATE SCHEMA public; GRANT ALL ON SCHEMA public TO ash; GRANT ALL ON SCHEMA public TO public;'

echo "== sql-only migrate via store.Open =="
E2E_DIR="${ASH_E2E_DATA_DIR:-$(mktemp -d)}"
export ASH_DATA_DIR="$E2E_DIR"
go test -tags=integration ./internal/store/ -run TestPostgresSQLSchemaModeE2E -count=1

echo "== doctor M3 (M3-08 sql schema; M3-04 skipped without dual-write migrate) =="
env -u ASH_MIGRATE_E2E go run ./cmd/cli doctor --suite M3

echo "== rag postgres fts integration test =="
go test -tags=integration ./internal/rag/ -count=1 -run TestPostgresRAGFTSQuery

echo "== postgres RLS memory child isolation =="
export ASH_POSTGRES_RLS=1
export ASH_POSTGRES_RLS_FORCE=1
bash scripts/postgres-ensure-app-role.sh
go test -tags=integration ./internal/store/ -run 'TestPostgresRLSPoliciesInstalled|TestPostgresRLSSpaceIsolationOnMemoryChildren|TestPostgresRLSSpaceIsolationOnOrgIdentity' -count=1

echo "== doctor TR3 on live postgres (TR3-06 fts; TR3-02 fallback sqlite-only) =="
env -u ASH_MIGRATE_E2E go run ./cmd/cli doctor --suite TR3 --agent static

echo "OK postgres sql-schema e2e (mode=sql, data dir: $E2E_DIR)"
