#!/usr/bin/env bash
# RLS + ash_app worker smoke on live Postgres (after migrate or standalone with reset).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
# shellcheck source=_go_env.sh
source "$ROOT/scripts/_go_env.sh"
_ash_go_env_bootstrap "$ROOT"

PORT="${ASH_POSTGRES_PORT:-5432}"
if docker ps --format '{{.Names}}' 2>/dev/null | grep -qx ash-postgres-dev; then
  mapped="$(docker port ash-postgres-dev 5432/tcp 2>/dev/null | head -1 | sed 's/.*://')"
  if [[ -n "$mapped" ]]; then
    PORT="$mapped"
  fi
fi
export ASH_POSTGRES_PORT="$PORT"
export ASH_DATABASE_URL="postgres://ash:ash@127.0.0.1:${PORT}/ash?sslmode=disable"
export ASH_DATABASE_APP_URL="postgres://ash_app:ash_app@127.0.0.1:${PORT}/ash?sslmode=disable"
export ASH_POSTGRES_RLS=1
export ASH_POSTGRES_RLS_FORCE=1

if ! docker ps --format '{{.Names}}' 2>/dev/null | grep -qx ash-postgres-dev; then
  echo "postgres container not running; start with: make postgres-up" >&2
  exit 2
fi
echo "using postgres port ${PORT}" >&2

echo "== ensure roles =="
bash scripts/postgres-ensure-app-role.sh

echo "== apply RLS on postgres (owner) =="
go test -tags=integration ./internal/store/ -run TestPostgresRLSPoliciesInstalled -count=1

echo "== RLS isolation (ash_rls_tester) =="
go test -tags=integration ./internal/store/ -run TestPostgresRLSSpaceIsolationOnMemoryRecords -count=1
go test -tags=integration ./internal/store/ -run TestPostgresRLSSpaceIsolationOnMemoryChildren -count=1
go test -tags=integration ./internal/store/ -run TestPostgresRLSSpaceIsolationOnOrgIdentity -count=1

if [[ "${ASH_MIGRATE_E2E:-}" == "1" ]]; then
  echo "== RLS on migrated postgres (ash_app) =="
  go test -tags=integration ./internal/store/ -run TestPostgresRLSE2EAfterMigrate -count=1
fi

echo "OK postgres RLS e2e (app URL=${ASH_DATABASE_APP_URL})"
