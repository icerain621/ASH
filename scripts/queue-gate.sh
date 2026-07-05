#!/usr/bin/env bash
# MVP §4: TTL queue consume + governance alert baseline on clean space.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
# shellcheck source=_go_env.sh
source "$ROOT/scripts/_go_env.sh"
_ash_go_env_bootstrap "$ROOT"

echo "== TTL queue consume baseline =="
go test ./internal/api/... -run 'TestTTLQueueConsumeBaseline|TestMemoryTTLQueueAndSweepAPI' -count=1

echo "== memory TTL unit smoke =="
go test ./internal/memory/... -run 'TestTTLQueue|TestClassifyTTL' -count=1

echo "== clean-space alert baseline =="
go test ./internal/alerts/... -run TestEvaluateCleanSpaceNoCriticalAlerts -count=1

echo "OK queue-gate"
