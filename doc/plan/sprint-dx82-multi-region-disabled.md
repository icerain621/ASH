# Sprint DX82 — 多区域关闭探测 / multiRegion=disabled

> **前置 / Prerequisite：** DX81；[`v4.3-release-scope.md`](v4.3-release-scope.md)  
> **状态 / Status：** ✅  
> **做法 / Approach：** `/readyz` 与 `GET /scale/readiness` 固定返回 `multiRegion=disabled`。不实现 Active-Active。  
> `/readyz` and scale readiness always report `multiRegion=disabled`. No Active-Active.

## 任务板 / Board

| ID | 任务 / Task | 状态 |
|----|-------------|------|
| DX82-1 | `config.MultiRegion()` 恒为 `disabled` | ✅ |
| DX82-2 | HealthResponse + ScaleReadiness 字段与 parity | ✅ |
| DX82-3 | OpenAPI / swagger / Scale 页一行 | ✅ |

## 验收 / Verify

```bash
go test ./internal/config/ ./internal/api/ -count=1 -run 'TestMultiRegion|TestHealthzAndReadyz|TestScaleReadiness$|TestAssertReadyzScaleParity'
go test ./internal/openapicheck/ -count=1
```
