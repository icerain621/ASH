# Sprint DX67 — 配额门禁 / Quota gate

> **方案 / Approach：** `bodyJson.quotas`；Create/Spawn fail-closed  
> **状态 / Status：** ✅  
> **设计 / Design：** [`../../docs/superpowers/specs/2026-09-19-v41-enterprise-agentic-design.md`](../../docs/superpowers/specs/2026-09-19-v41-enterprise-agentic-design.md)

## 任务板 / Board

| ID | 任务 / Task | 状态 |
|----|------|------|
| DX67-1 | `quotas` 解析/校验 | ✅ |
| DX67-2 | StricterQuotas helper | ✅ |
| DX67-3 | Create/Spawn + `SPACE_QUOTA_EXCEEDED` (409) | ✅ |
| DX67-4 | 单测 / tests | ✅ |

## 验收 / Verify

```bash
go test ./internal/spacepolicy/ ./internal/runs/ ./internal/apicodes/ -count=1 -run 'Quota|Quotas|Catalog'
```
