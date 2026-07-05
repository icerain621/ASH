#!/usr/bin/env bash
# Prefill release window evidence stub (MVP §8).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DATE_UTC="$(date -u +%Y-%m-%d)"
DEST="$ROOT/doc/evidence/release-window-${DATE_UTC}.md"
LATEST="$ROOT/doc/evidence/release-window-latest.md"

mkdir -p "$(dirname "$DEST")"
{
  echo "# 发布窗口记录（${DATE_UTC}）"
  echo
  echo "- 生成时间（UTC）：$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  echo "- Git：$(git -C "$ROOT" rev-parse --short HEAD) @ $(git -C "$ROOT" rev-parse --abbrev-ref HEAD)"
  echo "- 手册：[\`release-window-runbook.md\`](../checklists/release-window-runbook.md)"
  echo
  echo "## 门禁快照"
  echo
  echo "| 门禁 | 命令 | 状态 |"
  echo "|------|------|------|"
  echo "| MVP 签字 | \`make mvp-signoff\` | 待执行 |"
  echo "| 配置核对 | \`make config-env-gate\` | 待执行 |"
  echo "| 云 RDS | \`make cloud-acceptance\` | 待环境 |"
  echo "| Live Worker | \`make worker-local-gate\` | 可选 |"
  echo
  echo "## 值班 roster（填写）"
  echo
  echo "| 角色 | 姓名 | 联系方式 |"
  echo "|------|------|----------|"
  echo "| 发布负责人 | | |"
  echo "| 技术 on-call | | |"
  echo "| DBA / 平台 | | |"
  echo
  echo "## 切换检查"
  echo
  echo "- [ ] \`make data-backup\`"
  echo "- [ ] \`make pre-migrate-gate\`"
  echo "- [ ] \`make cloud-acceptance\`"
  echo "- [ ] \`make t0-alert-gate\`"
  echo "- [ ] \`make t1-metrics-gate\`（T+1）"
} >"$DEST"
cp "$DEST" "$LATEST"
echo "OK release-window-prefill: $DEST"
