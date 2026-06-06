#!/usr/bin/env bash
# Ensure ash_app / ash_rls_tester roles exist (idempotent). Requires Postgres + superuser/migrator URL.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

URL="${ASH_DATABASE_URL:-postgres://ash:ash@127.0.0.1:${ASH_POSTGRES_PORT:-5432}/ash?sslmode=disable}"

if docker ps --format '{{.Names}}' 2>/dev/null | grep -qx ash-postgres-dev; then
  docker exec -i ash-postgres-dev psql -U ash -d ash -v ON_ERROR_STOP=1 < scripts/postgres-init/01-ash-roles.sql
  echo "OK postgres roles via docker (ash-postgres-dev)"
  exit 0
fi

if command -v psql >/dev/null 2>&1; then
  psql "$URL" -v ON_ERROR_STOP=1 -f scripts/postgres-init/01-ash-roles.sql
  echo "OK postgres roles via psql"
  exit 0
fi

echo "postgres-ensure-app-role: need docker container ash-postgres-dev or psql in PATH" >&2
exit 2
