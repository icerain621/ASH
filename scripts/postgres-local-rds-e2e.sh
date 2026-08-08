#!/usr/bin/env bash
# Local Docker stand-in for cloud postgres-rds-e2e (H-01 dry-run without cloud-rds.env).
# Requires: ash-postgres-dev running (`make postgres-up`) and a SQLite source (default .ash/ash.db).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
# shellcheck source=_go_env.sh
source "$ROOT/scripts/_go_env.sh"
_ash_go_env_bootstrap "$ROOT"

if ! docker ps --format '{{.Names}}' 2>/dev/null | grep -qx ash-postgres-dev; then
  echo "postgres container not running; start with: make postgres-up" >&2
  exit 2
fi

PORT="${ASH_POSTGRES_PORT:-5432}"
if mapped="$(docker port ash-postgres-dev 5432/tcp 2>/dev/null | head -1 | sed 's/.*://')"; then
  [[ -n "$mapped" ]] && PORT="$mapped"
fi

export ASH_POSTGRES_PORT="$PORT"
export ASH_DATABASE_URL="${ASH_DATABASE_URL:-postgres://ash:ash@127.0.0.1:${PORT}/ash?sslmode=disable}"
export ASH_DATABASE_APP_URL="${ASH_DATABASE_APP_URL:-postgres://ash_app:ash_app@127.0.0.1:${PORT}/ash?sslmode=disable}"
export ASH_DATA_DIR="${ASH_DATA_DIR:-.ash}"
export ASH_SQLITE_PATH="${ASH_SQLITE_PATH:-$ASH_DATA_DIR/ash.db}"
export ASH_MIGRATE_E2E=1
export ASH_POSTGRES_RLS=1
export ASH_POSTGRES_RLS_FORCE=1
export ASH_SCHEMA_MODE="${ASH_SCHEMA_MODE:-sql}"

if [[ ! -f "$ASH_SQLITE_PATH" ]]; then
  echo "SQLite source missing at $ASH_SQLITE_PATH (bootstrap with make bootstrap-local-ash-db)" >&2
  exit 2
fi

echo "== postgres local RDS e2e (Docker port ${PORT}) =="
bash scripts/postgres-rds-e2e.sh
echo "OK postgres-local-rds-e2e"
