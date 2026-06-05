#!/usr/bin/env bash
# Start local Postgres for ASH migration dev/e2e.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

if ! command -v docker >/dev/null 2>&1; then
  echo "docker is required" >&2
  exit 2
fi

PORT="${ASH_POSTGRES_PORT:-5432}"
export ASH_POSTGRES_PORT="$PORT"

docker compose -f docker-compose.postgres.yml up -d --wait

POSTGRES_URL="postgres://ash:ash@127.0.0.1:${PORT}/ash?sslmode=disable"
echo "Postgres ready (container: ash-postgres-dev)"
echo "Port: ${PORT} (override with ASH_POSTGRES_PORT=<port>)"
echo "Run this in your shell when you want ASH to use Postgres as the primary database:"
echo "export ASH_DATABASE_URL='${POSTGRES_URL}'"
