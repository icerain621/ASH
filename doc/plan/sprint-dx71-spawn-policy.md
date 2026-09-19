# Sprint DX71 — spawn 策略 / Spawn policy

> **前置 / Prerequisite：** DX67  
> **状态 / Status：** ✅  
> **设计 / Design：** [`../../docs/superpowers/specs/2026-09-19-v41-enterprise-agentic-design.md`](../../docs/superpowers/specs/2026-09-19-v41-enterprise-agentic-design.md)

空间 `bodyJson.subRun` 在已配置时覆盖 Harness `maxDepth`；工具与请求取交集，危险工具仍拒绝。Spawn 响应与 `run.spawned` 带上深度与 token 代理。  
When set, space `bodyJson.subRun` overrides Harness `maxDepth`. Tools are the intersection of policy and request; dangerous tools stay rejected. Spawn response and `run.spawned` expose depth and the token proxy.

## 任务板 / Board

| ID | 任务 / Task | 状态 / Status |
|----|------|------|
| DX71-1 | `bodyJson.subRun`（maxDepth / allowedTools / tokenBudgetProxy） | ✅ |
| DX71-2 | Spawn 读 SpacePolicy 覆盖 Harness / Spawn overrides Harness | ✅ |
| DX71-3 | 响应/事件暴露策略与预算 / Response and event fields | ✅ |
| DX71-4 | Trajectory 薄可见 / Thin Trajectory hint | ✅ |

## 验收 / Verify

```bash
go test ./internal/runs/ ./internal/spacepolicy/ -count=1 -run 'SubRun|Spawn|Policy'
```
