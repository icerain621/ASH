# Sprint DX68 — 配额只读投影 / Quota projection

> **前置 / Pref：** DX67  
> **状态 / Status：** ✅  
> **设计 / Design：** [`../../docs/superpowers/specs/2026-09-19-v41-enterprise-agentic-design.md`](../../docs/superpowers/specs/2026-09-19-v41-enterprise-agentic-design.md)

## 任务板 / Board

| ID | 任务 / Task | 状态 |
|----|-------------|------|
| DX68-1 | `GET /api/v1/spaces/{id}/quotas` | ✅ |
| DX68-2 | OpenAPI 契约 · contract | ✅ |
| DX68-3 | Space 登记面板只读投影 · console card | ✅ |
| DX68-4 | 签字测 · tests | ✅ |

## 验收 / Verify

```bash
go test ./internal/api/ ./internal/spacepolicy/ -count=1 -run 'Quota|BuildQuota'
# FE: Space registry panel shows 配额 summary
```
