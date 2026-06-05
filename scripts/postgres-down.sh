#!/usr/bin/env bash
# Stop local Postgres used for ASH migration e2e.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

if ! command -v docker >/dev/null 2>&1; then
  echo "docker is required" >&2
  exit 2
fi

docker compose -f docker-compose.postgres.yml down -v
echo "Postgres stopped"
