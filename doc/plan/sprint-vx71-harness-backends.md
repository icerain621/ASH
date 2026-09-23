# Sprint VX71 — Harness 回显沙箱目录 / Harness shows sandbox catalog

> **前置 / Prerequisite：** v6.6 已冻结；[`v6.7-release-scope.md`](v6.7-release-scope.md)  
> **状态 / Status：** ✅  
> **做法 / Approach：** 列表与 active 回显 `sandbox.KnownBackendIDs()`。不改 spec、无新表。  

## 任务板 / Board

| ID | 任务 / Task | 状态 |
|----|-------------|------|
| VX71-1 | 列表与 active 响应 `sandboxBackends` | ✅ |
| VX71-2 | OpenAPI 对齐 | ✅ |
| VX71-3 | 控制台一行沙箱目录 | ✅ |

## 验收 / Verify

```bash
go test ./internal/api/ -count=1 -run 'TestHarnessProfileAPILifecycle'
```
