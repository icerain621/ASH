#!/usr/bin/env bash
# Verify MVP §11 signatures and scope freeze are complete before release tag.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

SIG="$ROOT/doc/evidence/mvp-signatures-latest.md"
ENV_FILE="${ASH_SIGNOFF_ENV:-$ROOT/config/signoff.env}"
FAIL=0

check_file_nonempty() {
  local file="$1"
  local pattern="$2"
  local label="$3"
  if [[ ! -f "$file" ]]; then
    echo "FAIL $label: missing $file" >&2
    FAIL=1
    return
  fi
  if ! grep -qE "$pattern" "$file"; then
    echo "FAIL $label: pattern not found in $file" >&2
    FAIL=1
    return
  fi
  echo "OK $label"
}

if [[ -f "$ENV_FILE" ]]; then
  set -a
  # shellcheck disable=SC1090
  source "$ENV_FILE"
  set +a
fi

for role in PRODUCT TECH QA RELEASE; do
  var="ASH_SIGNOFF_${role}_NAME"
  if [[ -z "${!var:-}" ]]; then
    echo "FAIL §11: ${var} empty (config/signoff.env)" >&2
    FAIL=1
  else
    echo "OK §11: ${var}=${!var}"
  fi
done

if [[ "${ASH_SCOPE_FREEZE_SIGNED:-0}" != "1" ]]; then
  echo "FAIL scope freeze: ASH_SCOPE_FREEZE_SIGNED!=1" >&2
  FAIL=1
else
  echo "OK scope freeze signed"
fi

if [[ -f "$SIG" ]]; then
  check_file_nonempty "$SIG" '^\| 产品负责人 \| [^|]+ \| [0-9]{4}-' 'mvp-signatures product row'
  check_file_nonempty "$SIG" '范围冻结：已冻结' 'mvp-signatures scope frozen'
else
  echo "FAIL mvp-signatures-latest.md missing (run make signoff-apply)" >&2
  FAIL=1
fi

SCOPE="$ROOT/doc/mvp-release-scope.md"
if [[ -f "$SCOPE" ]] && grep -q '已冻结' "$SCOPE"; then
  echo "OK mvp-release-scope.md marked frozen"
else
  echo "FAIL mvp-release-scope.md not marked 已冻结" >&2
  FAIL=1
fi

CHECKLIST="$ROOT/doc/11-mvp-release-checklist.md"
if grep -q '__________' "$CHECKLIST" 2>/dev/null; then
  echo "FAIL 11-mvp-release-checklist.md §11 still has placeholders" >&2
  FAIL=1
else
  echo "OK 11-mvp-release-checklist.md §11 filled"
fi

if [[ "$FAIL" -ne 0 ]]; then
  echo "" >&2
  echo "signoff-gate FAILED — see doc/checklists/mvp-signoff-roster.md" >&2
  exit 1
fi

echo "OK signoff-gate"
