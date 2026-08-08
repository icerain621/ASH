# 进度归属（progress）

**本目录跟踪「发布与验收进度」；日常开发完成度叙事仍在 [`../plan/PLAN-进度与里程碑.md`](../plan/PLAN-进度与里程碑.md)。**

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
