# Sprint VX42 — Ingress 探测投影 / Ingress adapters on probes

> **前置 / Prerequisite：** VX41  
> **状态 / Status：** ✅  
> **做法 / Approach：** `/readyz` 与 scale readiness 增加 `ingressAdapters`（投影 `KnownAdapterIDs`）。无新表。  

## 任务板 / Board

| ID | 任务 / Task | 状态 |
|----|-------------|------|
| VX42-1 | HealthResponse + ScaleReadiness + OpenAPI | ✅ |
| VX42-2 | 单测与 parity | ✅ |
| VX42-3 | CHANGELOG | ✅ |

## 验收 / Verify

```bash
make openapi-check
go test ./internal/api/ -count=1 -run 'Readyz|ScaleReadiness$|AssertReadyz'
```
