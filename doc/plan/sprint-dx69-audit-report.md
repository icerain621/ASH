# Sprint DX69 — 审计薄报表 API

> **前置：** DX67（可并行 DX68）  
> **原则：** 扫现有 `audit_log`，**无新表**  
> **状态：** ⬜  
> **设计：** [`../../docs/superpowers/specs/2026-09-19-v41-enterprise-agentic-design.md`](../../docs/superpowers/specs/2026-09-19-v41-enterprise-agentic-design.md)

## 任务板

| ID | 任务 | 状态 |
|----|------|------|
| DX69-1 | `GET …/spaces/{id}/audit-report?window=` 计数聚合 | ⬜ |
| DX69-2 | 覆盖 approve/deny/hook/spawn 等关键 event_type | ⬜ |
| DX69-3 | OpenAPI + 单测 | ⬜ |

## 验收

```bash
go test ./internal/api/ -count=1 -run 'AuditReport'
make openapi-check
```
