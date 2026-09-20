# Sprint VX44 — 组织 Skills Hub 标记 / Org skills hub marker

> **前置 / Prerequisite：** VX43  
> **状态 / Status：** ✅  
> **做法 / Approach：** `GET /skills/catalog` 增加 `hub=org`；控制台「私有 · 组织 Hub · 不计费」。无新表、无公网市场。  

## 任务板 / Board

| ID | 任务 / Task | 状态 |
|----|-------------|------|
| VX44-1 | 响应字段 + OpenAPI | ✅ |
| VX44-2 | 单测（缺省 catalog 仍带标记） | ✅ |
| VX44-3 | CHANGELOG | ✅ |

## 验收 / Verify

```bash
go test ./internal/skills/ -count=1 -run Catalog
make openapi-check
```
