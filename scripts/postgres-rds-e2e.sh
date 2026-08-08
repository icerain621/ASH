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

echo "== pre-migrate backup/plan (optional) =="
bash scripts/pre-migrate-gate.sh || true

echo "== golang-migrate schema (expected revision 20) =="
bash scripts/postgres-ensure-app-role.sh
go run ./cmd/cli migrate schema up --postgres "$ASH_DATABASE_URL"
go run ./cmd/cli migrate schema version --postgres "$ASH_DATABASE_URL"

echo "== sqlite → postgres data copy =="
if [[ "${ASH_DATABASE_URL}" == *"127.0.0.1"* || "${ASH_DATABASE_URL}" == *"localhost"* ]]; then
  if docker ps --format '{{.Names}}' 2>/dev/null | grep -qx ash-postgres-dev; then
    echo "== truncate local postgres data (dev e2e drift) =="
    docker exec -i ash-postgres-dev psql -U ash -d ash -v ON_ERROR_STOP=1 < scripts/postgres-truncate-dev-data.sql
  fi
fi

go run ./cmd/cli migrate plan  --data-dir "$ASH_DATA_DIR" --sqlite "$ASH_SQLITE_PATH" --postgres "$ASH_DATABASE_URL"
go run ./cmd/cli migrate copy  --data-dir "$ASH_DATA_DIR" --sqlite "$ASH_SQLITE_PATH" --postgres "$ASH_DATABASE_URL"
go run ./cmd/cli migrate verify --data-dir "$ASH_DATA_DIR" --sqlite "$ASH_SQLITE_PATH" --postgres "$ASH_DATABASE_URL"

echo "== doctor M3 (migrate verify + tenant isolation) =="
if [[ -n "${ASH_DATABASE_APP_URL:-}" ]]; then
  go run ./cmd/cli doctor --suite M3 --format md --require M3-04,M3-06,M3-07
else
  go run ./cmd/cli doctor --suite M3 --format md --require M3-04
fi

go test -tags=integration ./internal/store/ -run TestPostgresRLSE2EAfterMigrate -count=1

# M3/TR3 probes insert rows into Postgres; re-running M3-04 in later suites would false-fail
# (source SQLite unchanged). Migrate verify is already asserted above.
unset ASH_MIGRATE_E2E

go run ./cmd/cli doctor --suite TR3 --agent static --format md --require TR3-06,TR3-10

go run ./cmd/cli doctor --suite ALL --agent static --format md --require M3-06,M3-07,M3-09,M3-11,TR3-06,TR3-09,TR3-10

if [[ -n "${ASH_WORKER_URL:-}" ]]; then
  echo "== H-04..H-09 live smoke (ASH_WORKER_URL set) =="
  bash scripts/live-smoke.sh
else
  echo "== H-09 release sampling static =="
  bash scripts/release-sampling-static.sh
fi

echo "OK cloud RDS e2e (see doc/checklists/postgres-rds-e2e.md §7 manual SSE)"
