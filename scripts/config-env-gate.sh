#!/usr/bin/env bash
# MVP §8: aggregate production config / scope / template checks.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
EXAMPLE="$ROOT/config/cloud-rds.env.example"

echo "== cloud-rds.env.example required keys =="
for key in ASH_DATABASE_URL ASH_DATABASE_APP_URL ASH_SCHEMA_MODE ASH_POSTGRES_RLS ASH_POSTGRES_RLS_FORCE; do
  if ! grep -q "^${key}=" "$EXAMPLE"; then
    echo "missing $key in $EXAMPLE" >&2
    exit 1
  fi
done

echo "== production-config-gate =="
bash scripts/production-config-gate.sh

echo "== scope-freeze-gate =="
bash scripts/scope-freeze-gate.sh

if [[ -f "$ROOT/config/cloud-rds.env" ]]; then
  echo "== local cloud-rds.env placeholder scan =="
  if grep -E '^[^#].*CHANGE_ME' "$ROOT/config/cloud-rds.env"; then
    echo "FAIL unresolved CHANGE_ME in config/cloud-rds.env" >&2
    exit 1
  fi
  echo "OK cloud-rds.env has no CHANGE_ME placeholders"
else
  echo "skip cloud-rds.env (copy from config/cloud-rds.env.example)"
fi

echo "OK config-env-gate"
