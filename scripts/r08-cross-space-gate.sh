#!/usr/bin/env bash
# R-08 release-window cross-space / RLS sampling gate.
# Always runs API isolation regression; optionally Postgres RLS when Docker/URL ready.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
# shellcheck source=_go_env.sh
source "$ROOT/scripts/_go_env.sh"
_ash_go_env_bootstrap "$ROOT"

echo "== R-08 API cross-space regression =="
go test ./internal/api/ -run 'TestCrossSpaceAPIRegression|TestGetMemoryRecordRejectsCrossSpace|TestGetRunRejectsCrossSpace|TestSpaceMembersRejectsCrossSpaceParam|TestCreateRunRejectsCrossSpaceBody|TestRotateSecretRejectsCrossSpace|TestReleaseSamplingH09CrossSpaceMemoryDenied' -count=1

echo "== R-08 RLS catalog / SQL coverage (static) =="
go test ./internal/store/ -run 'TestMigrationCatalog_RLSCoverage|TestVerifyRLSMigrationSQL|TestRLSExpectedPolicyCount' -count=1

RLS_LIVE=0
if [[ "${ASH_R08_SKIP_POSTGRES:-0}" == "1" ]]; then
  echo "SKIP postgres RLS live (ASH_R08_SKIP_POSTGRES=1)"
elif [[ -n "${ASH_DATABASE_URL:-}" ]]; then
  echo "== R-08 Postgres RLS live (ASH_DATABASE_URL) =="
  go test ./internal/store/ -run 'TestPostgresRLS|TestRLS' -count=1 || {
    echo "NOTE: named RLS tests may be integration-tagged; trying postgres-rls-e2e" >&2
    bash scripts/postgres-rls-e2e.sh
  }
  RLS_LIVE=1
elif command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1; then
  echo "== R-08 Postgres RLS e2e (Docker) =="
  bash scripts/postgres-rls-e2e.sh
  RLS_LIVE=1
else
  echo "SKIP postgres RLS live (no ASH_DATABASE_URL / Docker)"
fi

echo "OK r08-cross-space-gate (api=ok rls_live=${RLS_LIVE})"
