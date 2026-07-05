#!/usr/bin/env bash
# MVP §5/§8: reject dev secrets and unresolved cloud-rds.env placeholders.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
# shellcheck source=_go_env.sh
source "$ROOT/scripts/_go_env.sh"
_ash_go_env_bootstrap "$ROOT"

echo "== production config guard (unit) =="
go test ./internal/config/... -run 'TestValidateProduction|TestEnvFilePlaceholder|TestProductionGuard' -count=1

ENV_FILE="${ASH_CLOUD_RDS_ENV:-$ROOT/config/cloud-rds.env}"
if [[ -f "$ENV_FILE" ]]; then
  echo "== scan $ENV_FILE for CHANGE_ME placeholders =="
  if grep -E '^[^#].*CHANGE_ME' "$ENV_FILE" >/tmp/ash-prod-placeholders.txt 2>/dev/null; then
    cat /tmp/ash-prod-placeholders.txt
    echo "FAIL unresolved placeholders in $ENV_FILE" >&2
    exit 1
  fi
  echo "OK no CHANGE_ME placeholders in $ENV_FILE"
else
  echo "skip env file scan ($ENV_FILE not present; copy config/cloud-rds.env.example)"
fi

echo "OK production-config-gate"
