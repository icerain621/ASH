# Sprint DX79 — 私有目录标记 / Private catalog marker

> **前置 / Prerequisite：** v4.2 已冻结；[`v4.3-release-scope.md`](v4.3-release-scope.md)  
> **状态 / Status：** ✅  
> **做法 / Approach：** `GET /skills/catalog` 固定返回 `marketplace=private`、`billing=none`。无新表、无市场页。  
> `GET /skills/catalog` always returns `marketplace=private` and `billing=none`. No new table and no marketplace page.

## 任务板 / Board

| ID | 任务 / Task | 状态 |
|----|-------------|------|
| DX79-1 | 响应字段（成功与失败路径） | ✅ |
| DX79-2 | OpenAPI + swagger | ✅ |
| DX79-3 | 单测：缺省 catalog 仍带标记 | ✅ |

## 验收 / Verify

```bash
go test ./internal/skills/ -count=1 -run TestLoadCatalogMissingOK
```
