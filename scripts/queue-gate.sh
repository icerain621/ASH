#!/usr/bin/env bash
# MVP §4: TTL queue consume + run backlog signal + governance alert baseline.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
# shellcheck source=_go_env.sh
source "$ROOT/scripts/_go_env.sh"
_ash_go_env_bootstrap "$ROOT"

echo "== TTL queue consume baseline =="
go test ./internal/api/... -run 'TestTTLQueueConsumeBaseline|TestMemoryTTLQueueAndSweepAPI' -count=1

echo "== run backlog (Scale) + cancel derive =="
go test ./internal/api/... -run 'TestScaleReadinessRunBacklogCounts' -count=1
go test ./internal/observability/derive/... -run 'TestReplay_runCanceledClearsInflight|TestCatalog_covers' -count=1

echo "== memory TTL unit smoke =="
go test ./internal/memory/... -run 'TestTTLQueue|TestClassifyTTL' -count=1

echo "== clean-space alert baseline =="
go test ./internal/alerts/... -run TestEvaluateCleanSpaceNoCriticalAlerts -count=1

echo "OK queue-gate"
