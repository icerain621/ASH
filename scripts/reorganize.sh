#!/usr/bin/env bash
# One-time layout normalization for ash monorepo
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

mkdir -p backend doc/product doc/design reports frontend scripts

# Move Go tree into backend/ (skip if already moved)
for item in cmd internal scenarios go.sum; do
  if [[ -e "$ROOT/$item" && ! -e "$ROOT/backend/$item" ]]; then
    mv "$ROOT/$item" "$ROOT/backend/"
    echo "moved $item -> backend/"
  fi
done

for item in go.mod Makefile; do
  if [[ -f "$ROOT/$item" && ! -f "$ROOT/backend/$item" ]]; then
    mv "$ROOT/$item" "$ROOT/backend/"
    echo "moved $item -> backend/"
  elif [[ -f "$ROOT/$item" && -f "$ROOT/backend/$item" && "$item" == "Makefile" ]]; then
    rm -f "$ROOT/$item"
    echo "removed duplicate root Makefile"
  fi
done

# Product docs (MVP planning set)
product_files=(
  "01-ai-product-research.md"
  "02-formal-prd-template.md"
  "03-ash-product-requirements-design.md"
  "04-api-event-state.md"
  "05-db-design-and-api-examples.md"
  "06-iteration-plan-10-weeks.md"
  "07-tech-stack-decision.md"
  "08-frontend-architecture.md"
  "09-backend-architecture-go.md"
  "10-engineering-setup.md"
  "11-mvp-release-checklist.md"
  "12-risk-register.md"
  "13-milestone-review-template.md"
  "14-kpi-dashboard-definition.md"
  "15-jira-epic-story-breakdown.md"
  "16-jira-import.csv"
  "17-jira-import-tasks.csv"
)
for f in "${product_files[@]}"; do
  if [[ -f "$ROOT/doc/$f" ]]; then
    mv "$ROOT/doc/$f" "$ROOT/doc/product/"
    echo "moved doc/$f -> doc/product/"
  fi
done

# Design docs (v0.1 implementation spec from repowiki)
for f in "01-PRD-需求文档.md" "02-HLD-总体设计.md" "03-ARCH-架构与技术选型.md" "04-PLAN-进度与里程碑.md"; do
  if [[ -f "$ROOT/doc/$f" ]]; then
    mv "$ROOT/doc/$f" "$ROOT/doc/design/"
    echo "moved doc/$f -> doc/design/"
  fi
done

if [[ -d "$ROOT/doc/appendices" && ! -d "$ROOT/doc/design/appendices" ]]; then
  mv "$ROOT/doc/appendices" "$ROOT/doc/design/"
  echo "moved doc/appendices -> doc/design/"
fi

if [[ -f "$ROOT/doc/AI商业分析报告-ash_repwiki.md" ]]; then
  mv "$ROOT/doc/AI商业分析报告-ash_repwiki.md" "$ROOT/reports/"
  echo "moved report -> reports/"
fi

touch frontend/.gitkeep
: > scripts/.gitkeep 2>/dev/null || true

echo ""
echo "=== $ROOT ==="
ls -1 "$ROOT"
echo ""
echo "=== backend ==="
ls -1 "$ROOT/backend" 2>/dev/null || true
echo ""
echo "=== doc ==="
find "$ROOT/doc" -maxdepth 2 -type d | sort
