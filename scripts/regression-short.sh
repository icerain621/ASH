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
go test ./internal/api/... -run 'TestHealthzAndReadyzSQLite|TestReadyzOpsSnapshot|TestReadyzIncludesRLSCatalogWhenEnabled|TestReadyzLiveGateHints|TestCISyncRunsWithFixture|TestSecretRotateRepoConnectionH07|TestHealthEndpointsLatencyBaseline|TestConcurrentRunsListBaseline|TestTTLQueueConsumeBaseline|TestCrossSpaceAPIRegression|TestStreamRunResumesFromQueryLastEventID|TestRunControlNegativeStatusCodes|TestScaleReadinessRunBacklogCounts|TestReleaseGovernanceAPI' -count=1
go test ./internal/runs/... -run 'TestCanTransitionMatrix|TestApplyRunStatusIllegal|TestCancelIdempotentAndFromRunning|TestFailRunDoesNotOverwriteCanceled|TestResumeNotResumable|TestObserveCanceled|TestMidLoopCancelStopsWithoutFinish|TestCanReplayMatrix|TestReplayRejectsNonTerminalSource|TestCanApproveCanResumeMatrix|TestApproveRejectsNonWaitingAndCanceled' -count=1
go test ./internal/observability/derive/... -run 'TestReplay_runCanceledClearsInflight|TestCatalog_covers' -count=1
go test ./internal/releases/... -run 'TestReleaseChecklistGateAndRollbackDrill' -count=1
go test ./internal/config/... -run 'TestValidateProduction|TestEnvFilePlaceholder|TestProductionGuard' -count=1
bash scripts/release-sampling-static.sh
go test ./internal/memory/... -run 'TestRunMigrations|TestDefaultTTLForLayer|TestEffectiveTTL|TestTTLQueue|TestClassifyTTL|TestApplyFeedbackDecayLowScore|TestQueryRanksByConfidenceAndFiltersFloor' -count=1
go test ./internal/ci/... -run 'TestFixtureProvider|TestDiagnoseLogClassifies|TestServiceSyncJobsDiagnose|TestGitHubProviderRetriesThenSucceeds|TestGitHubProviderCircuitOpensAfterFailures|TestGitHubProviderDoesNotRetryAuthErrors|TestIsRetryableGitHubError' -count=1
go test ./internal/store/... -run 'TestPostgresMigrationDSN|TestRuntimeDatabaseURL|TestVerifySQLiteFile' -count=1
go test ./internal/opsenv/... -count=1
go test ./internal/memory/... -count=1 -short
go test ./internal/openapicheck -run 'TestContractMatchesSwagger|TestApiV1SuccessResponsesAvoidGenericEnvelope|TestValidateContract|TestValidateReadyzContract' -count=1
go test ./internal/store -run 'TestMigrationCatalog_RLSCoverage|TestVerifyRLSMigrationSQL|TestRLSExpectedPolicyCount' -count=1
bash scripts/ash-data-backup-smoke.sh

echo "OK regression-short"
