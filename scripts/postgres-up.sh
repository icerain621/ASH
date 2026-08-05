#!/usr/bin/env bash
# Start local Postgres for ASH migration dev/e2e.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
# shellcheck source=_docker_compose.sh
source "$ROOT/scripts/_docker_compose.sh"

if ! command -v docker >/dev/null 2>&1; then
  echo "docker is required (install Docker Desktop / Engine, or set ASH_DATABASE_URL and ASH_SKIP_POSTGRES_UP=1)" >&2
  exit 2
fi

if [[ "${ASH_SKIP_POSTGRES_UP:-0}" == "1" ]]; then
  if [[ -z "${ASH_DATABASE_URL:-}" ]]; then
    echo "ASH_SKIP_POSTGRES_UP=1 requires ASH_DATABASE_URL" >&2
    exit 2
  fi
  echo "skip postgres-up; using ASH_DATABASE_URL=${ASH_DATABASE_URL}"
  if [[ "${BASH_SOURCE[0]}" != "${0}" ]]; then
    return 0
  fi
  exit 0
fi

echo "== docker diagnostics =="
docker version --format '{{.Server.Version}}' 2>/dev/null || docker version
docker compose version 2>/dev/null || true

PORT="${ASH_POSTGRES_PORT:-5432}"
if command -v powershell.exe >/dev/null 2>&1; then
  if powershell.exe -NoProfile -Command "Test-NetConnection -ComputerName 127.0.0.1 -Port $PORT -WarningAction SilentlyContinue | Select-Object -ExpandProperty TcpTestSucceeded" 2>/dev/null | grep -qi true; then
    if [[ "$PORT" == "5432" ]]; then
      PORT=5433
      echo "port 5432 busy; using ASH_POSTGRES_PORT=$PORT" >&2
    fi
  fi
fi
export ASH_POSTGRES_PORT="$PORT"
IMAGE="${ASH_POSTGRES_IMAGE:-postgres:16-alpine}"

if ! docker image inspect "$IMAGE" >/dev/null 2>&1; then
  if ! docker pull "$IMAGE" 2>/dev/null; then
    for fallback in postgres:15-alpine postgres:16-alpine; do
      if docker image inspect "$fallback" >/dev/null 2>&1; then
        IMAGE="$fallback"
        echo "using local image $IMAGE" >&2
        break
      fi
    done
  fi
fi

ASH_POSTGRES_IMAGE="$IMAGE" docker_compose -f docker-compose.postgres.yml up -d --wait

export ASH_DATABASE_URL="postgres://ash:ash@127.0.0.1:${PORT}/ash?sslmode=disable"
echo "ASH_DATABASE_URL=${ASH_DATABASE_URL}"
echo "Postgres ready (container: ash-postgres-dev)"
