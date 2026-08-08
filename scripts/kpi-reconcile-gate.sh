#!/usr/bin/env bash
# doc/plan/kpi-dashboard-definition.md §9: KPI overview vs derive replay reconciliation.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
# shellcheck source=_go_env.sh
source "$ROOT/scripts/_go_env.sh"
_ash_go_env_bootstrap "$ROOT"

echo "== KPI overview vs derive replay =="
go test ./internal/metrics/... -run 'TestKPIOverviewMatchesDeriveReplay|TestKPIOverviewSummaryCatalog' -count=1

echo "== derive replay parity =="
go test ./internal/observability/derive/... -run TestValidateReplayParity -count=1

echo "OK kpi-reconcile-gate"
