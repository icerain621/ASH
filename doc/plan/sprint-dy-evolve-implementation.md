# Sprint DY：演进平面基础 Implementation Plan

> **Goal:** feedback 全 targetType；`GET /reviews/queue`；`POST /reviews/{id}/decide`（memory + harness）。  
> **方案：** 已批准 A（薄演进 API）  
> **状态：** DY-1~DY-4 ✅

## 任务板

| ID | 任务 | 状态 |
|----|------|------|
| DY-1 | feedback 白名单 + runId + 唯一限流 | ✅ |
| DY-2 | `internal/evolve` 队列聚合 + decide | ✅ |
| DY-3 | API / OpenAPI / apicodes | ✅ |
| DY-4 | 单测 + CHANGELOG/TODO | ✅ |

## 退出（已满足）

- `harness_profile` feedback 可创建  
- 队列含 harness `in_review`（及 memory candidate）  
- decide approve → harness `active`  
