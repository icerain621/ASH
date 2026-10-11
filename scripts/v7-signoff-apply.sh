#!/usr/bin/env bash
# Apply four human signatures from config/signoff.env into doc/evidence/v7-signatures-latest.md.
# Requires a prior make v7-signoff-prep (or existing latest). Does NOT create a git tag.
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

ENV_FILE="${ASH_SIGNOFF_ENV:-$ROOT/config/signoff.env}"
OUT="$ROOT/doc/evidence/v7-signatures-latest.md"

if [[ ! -f "$ENV_FILE" ]]; then
  echo "signoff env not found: $ENV_FILE" >&2
  echo "hint: cp config/signoff.env.example config/signoff.env" >&2
  exit 2
fi
if [[ ! -f "$OUT" ]]; then
  echo "missing $OUT — run: make v7-signoff-prep" >&2
  exit 2
fi

set -a
# shellcheck disable=SC1090
source "$ENV_FILE"
set +a

missing=0
require() {
  local label="$1"
  local val="${2:-}"
  if [[ -z "$val" ]]; then
    echo "missing $label in signoff.env" >&2
    missing=1
  fi
}

require "ASH_SIGNOFF_PRODUCT_NAME" "${ASH_SIGNOFF_PRODUCT_NAME:-}"
require "ASH_SIGNOFF_PRODUCT_DATE" "${ASH_SIGNOFF_PRODUCT_DATE:-}"
require "ASH_SIGNOFF_TECH_NAME" "${ASH_SIGNOFF_TECH_NAME:-}"
require "ASH_SIGNOFF_TECH_DATE" "${ASH_SIGNOFF_TECH_DATE:-}"
require "ASH_SIGNOFF_QA_NAME" "${ASH_SIGNOFF_QA_NAME:-}"
require "ASH_SIGNOFF_QA_DATE" "${ASH_SIGNOFF_QA_DATE:-}"
require "ASH_SIGNOFF_RELEASE_NAME" "${ASH_SIGNOFF_RELEASE_NAME:-}"
require "ASH_SIGNOFF_RELEASE_DATE" "${ASH_SIGNOFF_RELEASE_DATE:-}"

if [[ "$missing" -ne 0 ]]; then
  echo "v7-signoff-apply aborted: fill config/signoff.env" >&2
  exit 1
fi

# Escape for sed replacement (|, &, \)
esc() {
  printf '%s' "$1" | sed -e 's/[\\|&]/\\&/g'
}

P_NAME="$(esc "$ASH_SIGNOFF_PRODUCT_NAME")"
P_DATE="$(esc "$ASH_SIGNOFF_PRODUCT_DATE")"
T_NAME="$(esc "$ASH_SIGNOFF_TECH_NAME")"
T_DATE="$(esc "$ASH_SIGNOFF_TECH_DATE")"
Q_NAME="$(esc "$ASH_SIGNOFF_QA_NAME")"
Q_DATE="$(esc "$ASH_SIGNOFF_QA_DATE")"
R_NAME="$(esc "$ASH_SIGNOFF_RELEASE_NAME")"
R_DATE="$(esc "$ASH_SIGNOFF_RELEASE_DATE")"

tmp="$(mktemp)"
# Replace empty signature table rows (name/date/sign columns blank).
sed -E \
  -e "s/^\\| 产品 \\| \\| \\| \\|$/| 产品 | ${P_NAME} | ${P_DATE} | ☑ |/" \
  -e "s/^\\| 技术 \\| \\| \\| \\|$/| 技术 | ${T_NAME} | ${T_DATE} | ☑ |/" \
  -e "s/^\\| 测试 \\| \\| \\| \\|$/| 测试 | ${Q_NAME} | ${Q_DATE} | ☑ |/" \
  -e "s/^\\| 发布 \\| \\| \\| \\|$/| 发布 | ${R_NAME} | ${R_DATE} | ☑ |/" \
  "$OUT" >"$tmp"

if ! grep -qE "^\| 产品 \| .+ \| .+ \| ☑ \|$" "$tmp"; then
  echo "FAIL could not apply product signature row (table shape changed?)" >&2
  rm -f "$tmp"
  exit 1
fi

mv "$tmp" "$OUT"
echo "OK applied v7 signatures to $OUT"
echo "Review the file, commit if appropriate, then optionally tag v7.0.0 by hand."
