# Sprint VX43 — Provider 目录标记 / Provider catalog markers

> **前置 / Prerequisite：** VX42  
> **状态 / Status：** ✅  
> **做法 / Approach：** `GET /model-router/providers` 响应信封固定 `catalog=org`、`billing=none`。无新表。  

## 任务板 / Board

| ID | 任务 / Task | 状态 |
|----|-------------|------|
| VX43-1 | 信封字段 + OpenAPI | ✅ |
| VX43-2 | 单测 | ✅ |
| VX43-3 | CHANGELOG | ✅ |

## 验收 / Verify

```bash
go test ./internal/api/ -count=1 -run 'ModelProvider|Providers'
make openapi-check
```
