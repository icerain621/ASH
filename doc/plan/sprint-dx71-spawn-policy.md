# Sprint DX71 — spawn 策略 / 预算可见

> **前置：** DX67  
> **状态：** ⬜  
> **设计：** [`../../docs/superpowers/specs/2026-09-19-v41-enterprise-agentic-design.md`](../../docs/superpowers/specs/2026-09-19-v41-enterprise-agentic-design.md)

## 任务板

| ID | 任务 | 状态 |
|----|------|------|
| DX71-1 | `bodyJson.subRun`（maxDepth / allowedTools / tokenBudgetProxy） | ⬜ |
| DX71-2 | Spawn 读 SpacePolicy 覆盖 Harness | ⬜ |
| DX71-3 | 响应/错误暴露策略与预算字段 | ⬜ |
| DX71-4 | Trajectory/控制台可见（薄） | ⬜ |

## 验收

```bash
go test ./internal/runs/ ./internal/spacepolicy/ -count=1 -run 'SubRun|Spawn|Policy'
```
