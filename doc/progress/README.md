# 进度归属（progress）

**本目录是发布勾选（标准类别：发布证据）。计划与完成度在 [`../plan/PLAN-进度与里程碑.md`](../plan/PLAN-进度与里程碑.md)。**  
**This folder holds release checklists. Plan narrative lives in PLAN.**

> 快照 2026-10-11：功能代 **v6.7 已冻结**；**v7 P0–P13 测试绿**（`make v7-signoff`）。发布侧：v7 待真人签字（`make v7-signoff-prep`）；云 RDS 仍待（见 [`../plan/GA-上线推进计划.md`](../plan/GA-上线推进计划.md)）。  
> Snapshot 2026-10-11: **v6.7 frozen**; **v7 P0–P13 green** (`make v7-signoff`). Release: v7 awaits human signatures (`make v7-signoff-prep`); cloud RDS still pending.

| 路径 | 用途 |
|------|------|
| [`mvp-release-checklist.md`](mvp-release-checklist.md) | MVP 发布总勾选（含签字位） |
| [`../checklists/`](../checklists/smoke-index.md) | 可执行烟测 / RDS / runbook（**路径稳定**） |
| [`../evidence/`](../evidence/README.md) | 门禁自动/半自动证据（**路径稳定**） |

## 工作流

```text
plan/TODO 标 P0
  → checklists 执行门禁
  → evidence 落 latest 报告
  → progress/mvp-release-checklist 勾选 + 签字
  → plan/PLAN 更新里程碑状态
```

## 脚本依赖的稳定路径

请勿移动 `doc/checklists/` 与 `doc/evidence/`。  
清单与范围文档路径：

- `doc/progress/mvp-release-checklist.md`
- `doc/plan/mvp-release-scope.md`
