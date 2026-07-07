#!/usr/bin/env bash
# H-09 static release sampling (postgres-rds-e2e.md §7 API paths, no Worker).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
# shellcheck source=_go_env.sh
source "$ROOT/scripts/_go_env.sh"
_ash_go_env_bootstrap "$ROOT"

PATTERN='TestReleaseSamplingH09|TestReleaseSamplingSSE|TestReleaseSamplingH09CrossSpaceMemoryDenied|TestReleaseSamplingCIFixtureH04H05'

echo "== H-09 release sampling static (§7 api tests) =="
env -u ASH_DATABASE_URL -u ASH_DATABASE_APP_URL -u ASH_MIGRATE_E2E \
  go test ./internal/api/... -run "$PATTERN" -count=1
echo "OK H-09 release sampling static"
