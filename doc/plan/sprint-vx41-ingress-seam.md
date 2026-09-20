# Sprint VX41 — Ingress 适配器缝 / Ingress adapter seam

> **前置 / Prerequisite：** v6.3 已冻结；[`v6.4-release-scope.md`](v6.4-release-scope.md)  
> **状态 / Status：** ✅  
> **做法 / Approach：** `internal/ingress`：`Adapter` + `null` / `webhook-github`。不引入 IM SDK。无新表。  

## 任务板 / Board

| ID | 任务 / Task | 状态 |
|----|-------------|------|
| VX41-1 | 包与接口 + 单测 | ✅ |
| VX41-2 | `KnownAdapterIDs()` 静态目录 | ✅ |
| VX41-3 | CHANGELOG | ✅ |

## 验收 / Verify

```bash
go test ./internal/ingress/... -count=1
```
