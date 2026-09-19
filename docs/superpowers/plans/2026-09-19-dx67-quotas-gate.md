# DX67 Implementation Plan · 配额门禁 / Quota gate

> 设计 / Spec: [`../specs/2026-09-19-v41-enterprise-agentic-design.md`](../specs/2026-09-19-v41-enterprise-agentic-design.md)  
> 板 / Board: [`../../../doc/plan/sprint-dx67-quotas-gate.md`](../../../doc/plan/sprint-dx67-quotas-gate.md)

## Files

| File | Role |
|------|------|
| `internal/spacepolicy/quotas.go` | Parse/validate `bodyJson.quotas` |
| `internal/runs/quota.go` | Count active runs; enforce before Create/Spawn |
| `internal/apicodes/catalog.go` | `SPACE_QUOTA_EXCEEDED` |
| `internal/api/handlers.go` / `subrun.go` | Map 409 |

## Tasks

1. Failing test: parse quotas + exceed concurrent  
2. Implement parse + enforce  
3. Wire Create/Spawn + API 409  
4. Register apicode · green tests · mark board  

## Out of this plan

DX68 projection · token usage accounting · DX71 subRun body.
