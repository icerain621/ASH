#!/usr/bin/env bash
# Stop local Postgres used for ASH migration e2e.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
# shellcheck source=_docker_compose.sh
source "$ROOT/scripts/_docker_compose.sh"

if ! command -v docker >/dev/null 2>&1; then
  echo "docker is required" >&2
  exit 2
fi

docker_compose -f docker-compose.postgres.yml down -v
echo "Postgres stopped"
