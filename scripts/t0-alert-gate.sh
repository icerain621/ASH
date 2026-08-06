#!/usr/bin/env bash
# MVP §10 T+0: governance alerts evaluate on clean DB (no active alerts).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
# shellcheck source=_go_env.sh
source "$ROOT/scripts/_go_env.sh"
_ash_go_env_bootstrap "$ROOT"

echo "== T+0 alert evaluate (clean space + inflight rule) =="
go test ./internal/alerts/... -run 'TestEvaluateCleanSpaceNoCriticalAlerts|TestEvaluateRunInflightAlert' -count=1

echo "== T+0 observability metrics smoke =="
go test ./internal/api/... -run TestObservabilityAlertsAndMetricsAPI -count=1

echo "OK t0-alert-gate"
