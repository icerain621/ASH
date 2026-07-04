#!/usr/bin/env bash
# H-08 static release audit (no cloud RDS; optional live Worker for §7).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
# shellcheck source=_go_env.sh
source "$ROOT/scripts/_go_env.sh"
_ash_go_env_bootstrap "$ROOT"

SKIP_OPENAPI="${ASH_RELEASE_AUDIT_SKIP_OPENAPI:-0}"
LIVE_WORKER="${ASH_WORKER_URL:-}"

echo "== H-08 release window audit (static) =="

echo "== §3 Doctor ALL (TestALLSuite, expect 43/43) =="
go test ./internal/doctor/... -run TestALLSuite -count=1

echo "== §3 Doctor M3 (TestM3Suite) =="
go test ./internal/doctor/... -run TestM3Suite -count=1

echo "== §3 Doctor TR3 (TestTR3Suite) =="
go test ./internal/doctor/... -run TestTR3Suite -count=1

if [[ -n "${ASH_RELEASE_AUDIT_DATA_DIR:-}" ]]; then
  echo "== §3 CLI doctor ALL md report (fresh ASH_DATA_DIR) =="
  ASH_DATA_DIR="$ASH_RELEASE_AUDIT_DATA_DIR" go run ./cmd/cli doctor --suite ALL --agent static --format md
else
  echo "== §3 CLI doctor md report skipped (set ASH_RELEASE_AUDIT_DATA_DIR for archive) =="
fi

echo "== §4 regression-short =="
go test ./internal/doctor/... -run 'TestM3Suite|TestTR3Suite|TestTR3PrometheusReplaySegmentWhenEnabled' -count=1
go test ./internal/alerts/... -count=1
go test ./internal/api/... -run 'TestHealthzAndReadyzSQLite|TestReadyzOpsSnapshot|TestReadyzIncludesRLSCatalogWhenEnabled|TestReadyzLiveGateHints|TestCISyncRunsWithFixture|TestReleaseSampling' -count=1
go test ./internal/memory/... -run 'TestRunMigrations|TestDefaultTTLForLayer|TestEffectiveTTL|TestTTLQueue|TestClassifyTTL' -count=1
go test ./internal/ci/... -run 'TestFixtureProvider|TestDiagnoseLogClassifiesTestFailure|TestServiceSyncJobsDiagnose' -count=1
go test ./internal/opsenv/... -count=1
go test ./internal/memory/... -count=1 -short
go test ./internal/openapicheck -run 'TestContractMatchesSwagger|TestApiV1SuccessResponsesAvoidGenericEnvelope|TestValidateContract|TestValidateReadyzContract' -count=1
go test ./internal/store -run 'TestMigrationCatalog_RLSCoverage|TestVerifyRLSMigrationSQL|TestRLSExpectedPolicyCount' -count=1

if [[ "$SKIP_OPENAPI" != "1" ]]; then
  echo "== §4 openapi-check =="
  if ! bash scripts/openapi-check.sh; then
    echo "openapi-check failed; retry with ASH_GOSUMDB_OFF=1 or ASH_RELEASE_AUDIT_SKIP_OPENAPI=1" >&2
    exit 1
  fi
else
  echo "== §4 openapi-check skipped (ASH_RELEASE_AUDIT_SKIP_OPENAPI=1) =="
fi

echo "== §7 release sampling (api tests) =="
go test ./internal/api/ -run 'TestReleaseSampling|TestCISyncRunsWithFixture|TestReleaseSamplingCIFixture' -count=1

echo "== H-06 M3-05 static (TestM3ExecGoLiveSmoke) =="
go test ./internal/doctor/... -run TestM3ExecGoLiveSmoke -count=1

if [[ "${ASH_EXECGO_E2E:-}" == "1" ]]; then
  echo "== H-06 ExecGo live smoke =="
  bash scripts/execgo-live-smoke.sh
else
  echo "== H-06 live ExecGo smoke skipped (set ASH_EXECGO_E2E=1) =="
fi

if [[ -n "$LIVE_WORKER" ]]; then
  echo "== §7 live release-sampling @ ${LIVE_WORKER} =="
  bash scripts/release-sampling.sh
  if curl -sf "${LIVE_WORKER}/readyz" | grep -q 'ASH_CI_FIXTURE'; then
    echo "== §7 CI fixture smoke =="
    bash scripts/ci-fixture-smoke.sh
  else
    echo "== §7 CI fixture smoke skipped (Worker without ASH_CI_FIXTURE) =="
  fi
else
  echo "== §7 live sampling skipped (set ASH_WORKER_URL for live curl checks) =="
fi

echo "OK H-08 static release audit"
