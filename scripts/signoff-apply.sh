#!/usr/bin/env bash
# Apply MVP §11 + scope freeze signatures from config/signoff.env to tracked docs.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

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
  echo "signoff-apply aborted: fill config/signoff.env (see doc/checklists/mvp-signoff-roster.md)" >&2
  exit 2
fi

GIT_SHA="$(git rev-parse --short HEAD 2>/dev/null || echo unknown)"
GIT_BRANCH="$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo unknown)"
NOW_UTC="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
DATE_UTC="$(date -u +%Y-%m-%d)"

SCOPE_FROZEN="草案"
SCOPE_STATUS_LINE="**草案 / 待产品评审签字**"
PRODUCT_CONCLUSION="范围确认（待冻结）"
if [[ "${ASH_SCOPE_FREEZE_SIGNED:-0}" == "1" ]]; then
  SCOPE_FROZEN="已冻结"
  freeze_date="${ASH_SCOPE_FREEZE_DATE:-$ASH_SIGNOFF_PRODUCT_DATE}"
  SCOPE_STATUS_LINE="**已冻结**（${freeze_date}，产品 \`${ASH_SIGNOFF_PRODUCT_NAME}\` 确认）"
  PRODUCT_CONCLUSION="范围确认 / 冻结"
fi

SIG_LATEST="$ROOT/doc/evidence/mvp-signatures-latest.md"
SIG_DATED="$ROOT/doc/evidence/mvp-signatures-${DATE_UTC}.md"
CHECKLIST="$ROOT/doc/progress/mvp-release-checklist.md"
SCOPE_DOC="$ROOT/doc/plan/mvp-release-scope.md"
MVP_LATEST="$ROOT/doc/evidence/mvp-signoff-latest.md"

write_signatures() {
  local dest="$1"
  {
    echo "# MVP v0.1 发布签字记录"
    echo
    echo "- 生成时间（UTC）：${NOW_UTC}"
    echo "- Git：${GIT_SHA} @ ${GIT_BRANCH}"
    echo "- 范围冻结：${SCOPE_FROZEN}（\`ASH_SCOPE_FREEZE_SIGNED=${ASH_SCOPE_FREEZE_SIGNED:-0}\`）"
    echo "- 操作：\`make signoff-apply\`"
    echo
    echo "## §11 四人签字"
    echo
    echo "| 角色 | 姓名 | 日期 |"
    echo "|------|------|------|"
    echo "| 产品负责人 | ${ASH_SIGNOFF_PRODUCT_NAME} | ${ASH_SIGNOFF_PRODUCT_DATE} |"
    echo "| 技术负责人 | ${ASH_SIGNOFF_TECH_NAME} | ${ASH_SIGNOFF_TECH_DATE} |"
    echo "| 测试负责人 | ${ASH_SIGNOFF_QA_NAME} | ${ASH_SIGNOFF_QA_DATE} |"
    echo "| 发布负责人 | ${ASH_SIGNOFF_RELEASE_NAME} | ${ASH_SIGNOFF_RELEASE_DATE} |"
    echo
    echo "## 范围冻结（§1）"
    echo
    echo "| 角色 | 姓名 | 日期 | 结论 |"
    echo "|------|------|------|------|"
    echo "| 产品 | ${ASH_SIGNOFF_PRODUCT_NAME} | ${ASH_SCOPE_FREEZE_DATE:-$ASH_SIGNOFF_PRODUCT_DATE} | ${PRODUCT_CONCLUSION} |"
    echo "| 技术 | ${ASH_SIGNOFF_TECH_NAME} | ${ASH_SIGNOFF_TECH_DATE} | 门禁达标 |"
    echo "| 测试 | ${ASH_SIGNOFF_QA_NAME} | ${ASH_SIGNOFF_QA_DATE} | 回归通过 |"
    echo
    if [[ -n "${ASH_RELEASE_WINDOW_START:-}" ]]; then
      echo "## 发布窗口（可选）"
      echo
      echo "- 窗口开始：${ASH_RELEASE_WINDOW_START}"
      echo "- 技术 on-call：${ASH_RELEASE_ONCALL_TECH:-}"
      echo "- DBA / 平台：${ASH_RELEASE_ONCALL_DBA:-}"
      echo
    fi
    echo "## 关联证据"
    echo
    echo "- [\`mvp-signoff-latest.md\`](mvp-signoff-latest.md)"
    echo "- [\`release-window-latest.md\`](release-window-latest.md)"
    echo "- [\`mvp-release-checklist.md\`](../progress/mvp-release-checklist.md) §11"
  } >"$dest"
}

write_signatures "$SIG_LATEST"
cp "$SIG_LATEST" "$SIG_DATED"

patch_line() {
  local file="$1"
  local old="$2"
  local new="$3"
  if grep -qF -- "$old" "$file"; then
    local old_escaped new_escaped
    old_escaped="$(printf '%s' "$old" | sed 's/[|/\\&]/\\&/g')"
    new_escaped="$(printf '%s' "$new" | sed 's/[|/\\&]/\\&/g')"
    sed -i "s|${old_escaped}|${new_escaped}|" "$file"
  elif grep -qF -- "$new" "$file"; then
    # Already applied (idempotent re-run) — silent
    :
  else
    echo "WARN: pattern not found in $file: $old" >&2
  fi
}

patch_line "$CHECKLIST" \
  '- 产品负责人：`__________`  日期：`____-__-__`' \
  "- 产品负责人：\`${ASH_SIGNOFF_PRODUCT_NAME}\`  日期：\`${ASH_SIGNOFF_PRODUCT_DATE}\`"
patch_line "$CHECKLIST" \
  '- 技术负责人：`__________`  日期：`____-__-__`' \
  "- 技术负责人：\`${ASH_SIGNOFF_TECH_NAME}\`  日期：\`${ASH_SIGNOFF_TECH_DATE}\`"
patch_line "$CHECKLIST" \
  '- 测试负责人：`__________`  日期：`____-__-__`' \
  "- 测试负责人：\`${ASH_SIGNOFF_QA_NAME}\`  日期：\`${ASH_SIGNOFF_QA_DATE}\`"
patch_line "$CHECKLIST" \
  '- 发布负责人：`__________`  日期：`____-__-__`' \
  "- 发布负责人：\`${ASH_SIGNOFF_RELEASE_NAME}\`  日期：\`${ASH_SIGNOFF_RELEASE_DATE}\`"

# Re-apply: update §11 lines when placeholders already filled
sed -i "s/^- 产品负责人：\`.*\`  日期：\`.*\`/- 产品负责人：\`${ASH_SIGNOFF_PRODUCT_NAME}\`  日期：\`${ASH_SIGNOFF_PRODUCT_DATE}\`/" "$CHECKLIST"
sed -i "s/^- 技术负责人：\`.*\`  日期：\`.*\`/- 技术负责人：\`${ASH_SIGNOFF_TECH_NAME}\`  日期：\`${ASH_SIGNOFF_TECH_DATE}\`/" "$CHECKLIST"
sed -i "s/^- 测试负责人：\`.*\`  日期：\`.*\`/- 测试负责人：\`${ASH_SIGNOFF_QA_NAME}\`  日期：\`${ASH_SIGNOFF_QA_DATE}\`/" "$CHECKLIST"
sed -i "s/^- 发布负责人：\`.*\`  日期：\`.*\`/- 发布负责人：\`${ASH_SIGNOFF_RELEASE_NAME}\`  日期：\`${ASH_SIGNOFF_RELEASE_DATE}\`/" "$CHECKLIST"

# mvp-release-scope.md — status line (line 3)
sed -i "3s|> 状态：.*|> 状态：${SCOPE_STATUS_LINE}（对应 [\`../progress/mvp-release-checklist.md\`](../progress/mvp-release-checklist.md) §1）  |" "$SCOPE_DOC"

# §6 table — idempotent rewrite (empty or previously filled)
perl -i -pe '
  s/^\| 产品 \| .* \| .* \| .* \|$/| 产品 | '"$ASH_SIGNOFF_PRODUCT_NAME"' | '"${ASH_SCOPE_FREEZE_DATE:-$ASH_SIGNOFF_PRODUCT_DATE}"' | '"$PRODUCT_CONCLUSION"' |/;
  s/^\| 技术 \| .* \| .* \| .* \|$/| 技术 | '"$ASH_SIGNOFF_TECH_NAME"' | '"$ASH_SIGNOFF_TECH_DATE"' | 门禁达标 |/;
  s/^\| 测试 \| .* \| .* \| .* \|$/| 测试 | '"$ASH_SIGNOFF_QA_NAME"' | '"$ASH_SIGNOFF_QA_DATE"' | 回归通过 |/;
' "$SCOPE_DOC"

if [[ -f "$MVP_LATEST" ]]; then
  perl -i -pe '
    s/^\| 产品负责人 \|.*$/| 产品负责人 | '"$ASH_SIGNOFF_PRODUCT_NAME"' | '"$ASH_SIGNOFF_PRODUCT_DATE"' |/;
    s/^\| 技术负责人 \|.*$/| 技术负责人 | '"$ASH_SIGNOFF_TECH_NAME"' | '"$ASH_SIGNOFF_TECH_DATE"' |/;
    s/^\| 测试负责人 \|.*$/| 测试负责人 | '"$ASH_SIGNOFF_QA_NAME"' | '"$ASH_SIGNOFF_QA_DATE"' |/;
    s/^\| 发布负责人 \|.*$/| 发布负责人 | '"$ASH_SIGNOFF_RELEASE_NAME"' | '"$ASH_SIGNOFF_RELEASE_DATE"' |/;
  ' "$MVP_LATEST"
  dated_mvp="$ROOT/doc/evidence/mvp-signoff-${DATE_UTC}.md"
  [[ -f "$dated_mvp" ]] && cp "$MVP_LATEST" "$dated_mvp" || true
fi

if [[ "${ASH_SCOPE_FREEZE_SIGNED:-0}" == "1" ]]; then
  patch_line "$CHECKLIST" \
    '- [x] 本次发布目标、范围、不包含范围已在评审中确认（[`mvp-release-scope.md`](../plan/mvp-release-scope.md) 草案，待签字）' \
    '- [x] 本次发布目标、范围、不包含范围已确认并冻结（[`mvp-release-scope.md`](../plan/mvp-release-scope.md)；`make signoff-gate`）'
fi

RUNBOOK="$ROOT/doc/checklists/release-window-runbook.md"
if [[ -f "$RUNBOOK" && -n "${ASH_RELEASE_WINDOW_START:-}" ]]; then
  window_display="${ASH_RELEASE_WINDOW_START}"
  if [[ "$window_display" =~ ^([0-9]{4}-[0-9]{2}-[0-9]{2})T([0-9]{2}):([0-9]{2}) ]]; then
    window_display="${BASH_REMATCH[1]} ${BASH_REMATCH[2]}:${BASH_REMATCH[3]}"
  fi
  export RB_WINDOW="$window_display"
  export RB_RELEASE_NAME="$ASH_SIGNOFF_RELEASE_NAME"
  export RB_RELEASE_DATE="$ASH_SIGNOFF_RELEASE_DATE"
  export RB_TECH_ONCALL="${ASH_RELEASE_ONCALL_TECH:-}"
  export RB_DBA="${ASH_RELEASE_ONCALL_DBA:-}"
  export RB_QA_NAME="$ASH_SIGNOFF_QA_NAME"
  export RB_TECH_DATE="$ASH_SIGNOFF_TECH_DATE"
  perl -i -pe '
    s/\r$//;
    if (/^\| 窗口开始（UTC\+8） \|/) {
      s/^\| 窗口开始（UTC\+8） \| [^|]+ \|$/| 窗口开始（UTC+8） | $ENV{RB_WINDOW} |/;
    }
    $in_roster = 1 if /^## 2\. 值班 roster/;
    $in_roster = 0 if /^## 3\./;
    $in_signoff = 1 if /^## 7\. 签字/;
    if ($in_roster && /^\| 发布负责人 \|/) {
      s{^\| 发布负责人 \|.*$}{| 发布负责人 | $ENV{RB_RELEASE_NAME} | | |};
    }
    if ($in_roster && /^\| 技术 on-call \|/) {
      s{^\| 技术 on-call \|.*$}{| 技术 on-call | $ENV{RB_TECH_ONCALL} | | |};
    }
    if ($in_roster && /^\| DBA \/ 平台 \|/) {
      s{^\| DBA / 平台 \|.*$}{| DBA / 平台 | $ENV{RB_DBA} | | |};
    }
    if ($in_roster && /^\| 测试 \|/) {
      s{^\| 测试 \|.*$}{| 测试 | $ENV{RB_QA_NAME} | | |};
    }
    if ($in_signoff && /^\| 发布负责人 \|/) {
      s{^\| 发布负责人 \|.*$}{| 发布负责人 | $ENV{RB_RELEASE_NAME} | $ENV{RB_RELEASE_DATE} |};
    }
    if ($in_signoff && /^\| 值班 SRE \|/) {
      s{^\| 值班 SRE \|.*$}{| 值班 SRE | $ENV{RB_TECH_ONCALL} | $ENV{RB_TECH_DATE} |};
    }
  ' "$RUNBOOK"
  echo "  runbook: $RUNBOOK (§1 window + §2 roster)"
fi

echo "OK signoff-apply"
echo "  signatures: $SIG_LATEST"
echo "  scope: $SCOPE_FROZEN"
if [[ "${ASH_SCOPE_FREEZE_SIGNED:-0}" != "1" ]]; then
  echo "WARN: set ASH_SCOPE_FREEZE_SIGNED=1 after product freeze confirmation"
fi
