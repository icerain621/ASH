# Sprint DX52 — 多入口统一 Plan（v3.1 · T2）

> **方案 A：** webhook 失败 → `FromGoal`（关键词前缀 hotfix）；`autoRun` → `AutoApprove`；**无新表**  
> **状态：** ✅ 完成  
> **设计：** [`../../docs/superpowers/specs/2026-09-08-v31-dx52-multi-entry-plan-design.md`](../../docs/superpowers/specs/2026-09-08-v31-dx52-multi-entry-plan-design.md)

## 任务板

| ID | 任务 | 状态 |
|----|------|------|
| DX52-1 | webhook → FromGoal（draft / autoApprove） | ✅ |
| DX52-2 | 响应 `planId` + 测试 | ✅ |
| DX52-3 | OpenAPI + sprint / TODO / CHANGELOG | ✅ |

## 验收

```bash
go test ./internal/api/ -run TestGitHubWebhook -count=1
# ok
make openapi-check
```

## 交付摘要

- CI 失败 → draft GoalPlan（Quest 可见）；`autoRun=1` 自动批准启动 hotfix Run
- 响应 `planId`（+ `ashRunId`/`ashTraceId` 当 autoApprove）
- Session 本就走 `FromGoal`，无行为变更
