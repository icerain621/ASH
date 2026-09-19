# Sprint DX68 — 配额只读投影

> **前置：** DX67  
> **状态：** ⬜  
> **设计：** [`../../docs/superpowers/specs/2026-09-19-v41-enterprise-agentic-design.md`](../../docs/superpowers/specs/2026-09-19-v41-enterprise-agentic-design.md)

## 任务板

| ID | 任务 | 状态 |
|----|------|------|
| DX68-1 | `GET /api/v1/spaces/{id}/quotas`（limit + usage） | ⬜ |
| DX68-2 | OpenAPI | ⬜ |
| DX68-3 | Space/控制台只读投影 | ⬜ |
| DX68-4 | 签字测 | ⬜ |

## 验收

```bash
make openapi-check
# FE: Space 页可见配额卡片（未配置显示「不限」）
```
