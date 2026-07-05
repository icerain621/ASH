#!/usr/bin/env bash
# Cloud RDS E2E: runs doc/checklists/postgres-rds-e2e.md appendix A (no Docker required).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
# shellcheck source=_go_env.sh
source "$ROOT/scripts/_go_env.sh"
_ash_go_env_bootstrap "$ROOT"

if [[ -z "${ASH_DATABASE_URL:-}" ]]; then
  echo "ASH_DATABASE_URL is required (migrator/owner URL)" >&2
  echo "See doc/checklists/postgres-rds-e2e.md" >&2
  exit 2
fi

export ASH_DATA_DIR="${ASH_DATA_DIR:-.ash}"
export ASH_SQLITE_PATH="${ASH_SQLITE_PATH:-$ASH_DATA_DIR/ash.db}"
export ASH_MIGRATE_E2E="${ASH_MIGRATE_E2E:-1}"
export ASH_POSTGRES_RLS="${ASH_POSTGRES_RLS:-1}"
export ASH_POSTGRES_RLS_FORCE="${ASH_POSTGRES_RLS_FORCE:-1}"
export ASH_SCHEMA_MODE="${ASH_SCHEMA_MODE:-sql}"

echo "== postgres RDS e2e (checklist appendix A) =="
echo "ASH_DATABASE_URL=${ASH_DATABASE_URL}"
echo "ASH_DATABASE_APP_URL=${ASH_DATABASE_APP_URL:-<unset>}"
echo "ASH_SQLITE_PATH=${ASH_SQLITE_PATH}"
echo "ASH_SCHEMA_MODE=${ASH_SCHEMA_MODE}"

bash scripts/postgres-smoke.sh

echo "== golang-migrate schema (expected revision 20) =="
go run ./cmd/cli migrate schema up --postgres "$ASH_DATABASE_URL"
go run ./cmd/cli migrate schema version --postgres "$ASH_DATABASE_URL"

bash scripts/postgres-ensure-app-role.sh

go run ./cmd/cli migrate plan  --data-dir "$ASH_DATA_DIR" --sqlite "$ASH_SQLITE_PATH" --postgres "$ASH_DATABASE_URL"
go run ./cmd/cli migrate copy  --data-dir "$ASH_DATA_DIR" --sqlite "$ASH_SQLITE_PATH" --postgres "$ASH_DATABASE_URL"
go run ./cmd/cli migrate verify --data-dir "$ASH_DATA_DIR" --sqlite "$ASH_SQLITE_PATH" --postgres "$ASH_DATABASE_URL"

go run ./cmd/cli doctor --suite M3 --format md
bash scripts/postgres-doctor-assert.sh M3 M3-04

go test -tags=integration ./internal/store/ -run TestPostgresRLS -count=1

if [[ -n "${ASH_DATABASE_APP_URL:-}" ]]; then
  bash scripts/postgres-doctor-assert.sh M3 M3-06,M3-07
fi

go run ./cmd/cli doctor --suite TR3 --format md
bash scripts/postgres-doctor-assert.sh TR3 TR3-06,TR3-10

go run ./cmd/cli doctor --suite ALL --agent static --format md

if [[ -n "${ASH_WORKER_URL:-}" ]]; then
  echo "== H-04..H-09 live smoke (ASH_WORKER_URL set) =="
  bash scripts/live-smoke.sh
else
  echo "== H-09 release sampling static =="
  bash scripts/release-sampling-static.sh
fi

echo "OK cloud RDS e2e (see doc/checklists/postgres-rds-e2e.md §7 manual SSE)"
