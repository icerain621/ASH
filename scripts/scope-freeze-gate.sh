#!/usr/bin/env bash
# MVP §1 + v2 + v2.1 + v2.2 + v2.3 + v2.4 + v2.5 + v2.6: validate release scope doc structure before freeze sign-off.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

required=(
  "## 1. 发布目标"
  "## 2. 包含范围"
  "## 3. 不包含范围"
  "## 4. 变更冻结规则"
  "## 5. 验收门禁"
  "## 6. 评审签字"
)

check_scope() {
  local scope="$1"
  local label="$2"
  if [[ ! -f "$scope" ]]; then
    echo "missing $scope" >&2
    exit 2
  fi
  for heading in "${required[@]}"; do
    if ! grep -qF "$heading" "$scope"; then
      echo "$label scope doc missing section: $heading ($scope)" >&2
      exit 1
    fi
  done
  if grep -q '待产品评审签字' "$scope" && [[ "${ASH_SCOPE_FREEZE_SIGNED:-0}" != "1" ]] && ! grep -q '已冻结' "$scope"; then
    echo "WARN $label scope doc still marked draft (set ASH_SCOPE_FREEZE_SIGNED=1 after product sign-off)"
  fi
  if grep -q '已冻结' "$scope"; then
    echo "OK $label scope marked 已冻结"
  else
    echo "WARN $label scope not yet marked 已冻结"
  fi
}

check_scope "$ROOT/doc/plan/mvp-release-scope.md" "mvp"
check_scope "$ROOT/doc/plan/v2-release-scope.md" "v2"
check_scope "$ROOT/doc/plan/v2.1-release-scope.md" "v2.1"
check_scope "$ROOT/doc/plan/v2.2-release-scope.md" "v2.2"
check_scope "$ROOT/doc/plan/v2.3-release-scope.md" "v2.3"
check_scope "$ROOT/doc/plan/v2.4-release-scope.md" "v2.4"
check_scope "$ROOT/doc/plan/v2.5-release-scope.md" "v2.5"
check_scope "$ROOT/doc/plan/v2.6-release-scope.md" "v2.6"

echo "OK scope-freeze-gate"
