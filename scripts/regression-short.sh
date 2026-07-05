#!/usr/bin/env bash
# Quick regression gate (Doctor M3/TR3, API, memory, CI fixture, openapicheck smoke).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
# shellcheck source=_go_env.sh
source "$ROOT/scripts/_go_env.sh"
_ash_go_env_bootstrap "$ROOT"

go test ./internal/doctor/... -run 'TestM3Suite|TestTR3Suite|TestTR3PrometheusReplaySegmentWhenEnabled|TestM3ExecGoLiveSmoke' -count=1
go test ./internal/alerts/... -count=1
go test ./internal/api/... -run 'TestHealthzAndReadyzSQLite|TestReadyzOpsSnapshot|TestReadyzIncludesRLSCatalogWhenEnabled|TestReadyzLiveGateHints|TestCISyncRunsWithFixture|TestReleaseSampling|TestSecretRotateRepoConnectionH07' -count=1
go test ./internal/memory/... -run 'TestRunMigrations|TestDefaultTTLForLayer|TestEffectiveTTL|TestTTLQueue|TestClassifyTTL' -count=1
go test ./internal/ci/... -run 'TestFixtureProvider|TestDiagnoseLogClassifiesTestFailure|TestServiceSyncJobsDiagnose' -count=1
go test ./internal/opsenv/... -count=1
go test ./internal/memory/... -count=1 -short
go test ./internal/openapicheck -run 'TestContractMatchesSwagger|TestApiV1SuccessResponsesAvoidGenericEnvelope|TestValidateContract|TestValidateReadyzContract' -count=1
go test ./internal/store -run 'TestMigrationCatalog_RLSCoverage|TestVerifyRLSMigrationSQL|TestRLSExpectedPolicyCount' -count=1

echo "OK regression-short"
