#!/usr/bin/env bash
# MVP §10 T+1: KPI overview + feedback ingest API baselines.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
# shellcheck source=_go_env.sh
source "$ROOT/scripts/_go_env.sh"
_ash_go_env_bootstrap "$ROOT"

echo "== T+1 KPI overview baseline =="
go test ./internal/api/... -run TestT1KPIOverviewBaseline -count=1
go test ./internal/metrics/... -run TestOverviewAggregatesKPIInputs -count=1

echo "== T+1 feedback baseline =="
go test ./internal/api/... -run TestT1FeedbackIngestBaseline -count=1

echo "OK t1-metrics-gate"
