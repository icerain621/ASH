#!/usr/bin/env bash
# Source cloud RDS env file (copy from config/cloud-rds.env.example first).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENV_FILE="${1:-$ROOT/config/cloud-rds.env}"

if [[ ! -f "$ENV_FILE" ]]; then
  echo "Env file not found: $ENV_FILE" >&2
  echo "Copy template: cp config/cloud-rds.env.example config/cloud-rds.env" >&2
  exit 2
fi

set -a
# shellcheck disable=SC1090
source "$ENV_FILE"
set +a

echo "Sourced $ENV_FILE"
echo "ASH_DATABASE_URL=${ASH_DATABASE_URL%%@*}@... (redacted)"
echo "ASH_DATABASE_APP_URL=${ASH_DATABASE_APP_URL:+set}"
echo "ASH_SCHEMA_MODE=${ASH_SCHEMA_MODE:-<unset>}"
