#!/usr/bin/env bash
# MVP §1: validate release scope doc structure before freeze sign-off.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SCOPE="$ROOT/doc/mvp-release-scope.md"

if [[ ! -f "$SCOPE" ]]; then
  echo "missing $SCOPE" >&2
  exit 2
fi

required=(
  "## 1. 发布目标"
  "## 2. 包含范围"
  "## 3. 不包含范围"
  "## 4. 变更冻结规则"
  "## 5. 验收门禁"
  "## 6. 评审签字"
)

for heading in "${required[@]}"; do
  if ! grep -qF "$heading" "$SCOPE"; then
    echo "scope doc missing section: $heading" >&2
    exit 1
  fi
done

if grep -q '待产品评审签字' "$SCOPE" && [[ "${ASH_SCOPE_FREEZE_SIGNED:-0}" != "1" ]] && ! grep -q '已冻结' "$SCOPE"; then
  echo "WARN scope doc still marked draft (set ASH_SCOPE_FREEZE_SIGNED=1 after product sign-off)"
fi

echo "OK scope-freeze-gate"
