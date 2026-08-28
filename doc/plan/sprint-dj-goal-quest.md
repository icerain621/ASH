# Sprint DJ：Goal→Run + Plan + 轻量 Quest UI Implementation Plan

> **方案：** 已批准 **C** = B（from-goal + Plan 审批闸门）+ Runs 页轻量 Quest 表单  
> **Goal:** NL Goal 路由到三场景，人工批准 Plan 后再 Create Run；CLI `ash quest`；Runs 页可从目标创建。  
> **For agentic workers:** 按任务板顺序实现；每任务结束后跑对应测试。

**Architecture:** `internal/goal` 负责关键词路由与填槽；`goal_plans` 持久化草稿；批准后调用既有 `runs.Create`。不改动 Run 状态机。

**Tech Stack:** Go / Gin / GORM / SQLite+Postgres SQL 24 / React RunsPage

## Global Constraints

- 三场景：`feature_delivery` / `hotfix` / `security_patch`（取 loader 最新版本）
- Plan 未批准不得启动 Run
- SQL rev **24** · RLS **44**
- OpenAPI + swagger + `make openapi-check`

---

> **状态：** ✅ 完成（2026-08-29）

## 任务板

| ID | 任务 | 状态 |
|----|------|------|
| DJ-1 | `internal/goal` 路由/填槽 + `goal_plans` + from-goal/approve/reject API | ✅ |
| DJ-2 | `ash quest` CLI + plan.* 事件 + openapi | ✅ |
| DJ-3 | RunsPage Quest 表单 + vitest | ✅ |
| DJ-4 | CHANGELOG / TODO / v2 计划 / 回归 | ✅ |

## API

| Method | Path | 说明 |
|--------|------|------|
| POST | `/api/v1/runs/from-goal` | body: `{goal, repoRoot?, spaceId?, autoApprove?}` → PlanView |
| GET | `/api/v1/runs/plans/{planId}` | 取草稿 |
| POST | `/api/v1/runs/plans/{planId}/approve` | → CreateResponse（启动 Run） |
| POST | `/api/v1/runs/plans/{planId}/reject` | 拒绝 |

`autoApprove=true`：跳过闸门直接 Create（CLI `--yes` / 测试用）。

## 路由规则（关键词，可后续换 LLM）

| 优先 | 关键词 | 场景 |
|------|--------|------|
| 1 | security, cve, vuln, 漏洞, 安全 | security_patch |
| 2 | hotfix, urgent, prod, 热修, 线上 | hotfix |
| 3 | 默认 | feature_delivery |

填槽：`issueOrSpec=goal`，`repoRoot` 默认 `.`。

## 退出标准

- [ ] from-goal 三场景路由单测绿
- [ ] approve 后产生 run；未 approve 无 run
- [ ] `ash quest "..." --yes` 可跑通
- [ ] Runs 页可提交 Goal、预览 Plan、批准
- [ ] openapi-check 绿
