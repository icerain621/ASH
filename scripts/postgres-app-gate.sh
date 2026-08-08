#!/usr/bin/env bash
# H-02/H-03 local Postgres ash_app + RLS gate (requires docker + migrated schema).
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
export ASH_DATABASE_URL="postgres://ash:ash@127.0.0.1:${PORT}/ash?sslmode=disable"
export ASH_DATABASE_APP_URL="postgres://ash_app:ash_app@127.0.0.1:${PORT}/ash?sslmode=disable"
export ASH_POSTGRES_RLS=1
export ASH_POSTGRES_RLS_FORCE=1

echo "== H-02/H-03 postgres app gate (port ${PORT}) =="

echo "== ensure ash_app role + schema rev 20 =="
bash scripts/postgres-ensure-app-role.sh
go run ./cmd/cli migrate schema up --postgres "$ASH_DATABASE_URL"

if ! docker exec ash-postgres-dev psql -U ash -d ash -tAc "SELECT 1 FROM information_schema.tables WHERE table_name='runs' LIMIT 1" | grep -q 1; then
  echo "postgres schema missing runs table after migrate schema up" >&2
  exit 2
fi

echo "== RLS ash_app smoke (existing schema, no reset) =="
export ASH_MIGRATE_E2E=1
go test -tags=integration ./internal/store/ -run TestPostgresRLSE2EAfterMigrate -count=1
# M3-04 needs a real SQLite source; this gate focuses H-02/H-03 (RLS + ash_app).
# Full migrate verify stays in postgres-rds-e2e / cloud-acceptance.
unset ASH_MIGRATE_E2E

GATE_DATA="${ASH_DATA_DIR:-}"
if [[ -z "$GATE_DATA" ]]; then
  GATE_DATA="$(mktemp -d)"
  trap 'rm -rf "$GATE_DATA"' EXIT
fi
export ASH_DATA_DIR="$GATE_DATA"

echo "== doctor M3-06/07 (ash_app runtime) =="
go run ./cmd/cli doctor --suite M3 --agent static --require M3-06,M3-07

echo "== H-03 readyz + RLS contract =="
go test -tags=integration ./internal/api/ -run 'TestPostgresReadyzWithRLS|TestPostgresReadyzScaleParity' -count=1

echo "OK postgres app gate (H-02/H-03 local)"
