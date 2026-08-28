# Sprint DZ：编排评审 UI + promote 硬化 Implementation Plan

> **方案：** 已批准 **B** = A（评审页 + promote 闸门）+ Scenario patch 草稿 UI  
> **Goal:** 控制台可编排评审；draft 不可直通 promote；rollback；scenario_patch 可进队列  
> **状态：** ✅ 完成（2026-08-28）

## 任务板

| ID | 任务 | 状态 |
|----|------|------|
| DZ-1 | Promote 仅 in_review；Rollback API | ✅ |
| DZ-2 | scenario_patch 表 + evolve 队列 | ✅ |
| DZ-3 | ReviewsPage + Automation Profile | ✅ |
| DZ-4 | Feedback 类型 + vitest + docs | ✅ |

## 验收要点

- `POST /harness/profiles/{id}/promote`：draft → 400；须 `in_review`（且 Reviews 批准后）
- `POST /harness/profiles/{id}/rollback`
- `GET/POST /scenario-patches` + `submit-review`；队列 id `scenario_patch:<id>`
- UI：`/ui/reviews`、Automation Harness 面板、Feedback targetType 含附录 K
- SQL rev **23** · RLS **43**
