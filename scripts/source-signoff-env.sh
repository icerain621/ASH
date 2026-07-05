#!/usr/bin/env bash
# Source config/signoff.env — must be dotted, not executed:
#   source scripts/source-signoff-env.sh
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENV_FILE="${ASH_SIGNOFF_ENV:-$ROOT/config/signoff.env}"

if [[ ! -f "$ENV_FILE" ]]; then
  echo "signoff env not found: $ENV_FILE" >&2
  echo "hint: cp config/signoff.env.example config/signoff.env" >&2
  exit 2
fi

set -a
# shellcheck disable=SC1090
source "$ENV_FILE"
set +a

echo "OK sourced signoff env from $ENV_FILE"
