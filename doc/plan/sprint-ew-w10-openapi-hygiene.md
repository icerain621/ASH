# Sprint EW · OpenAPI 卫生

> 前置：EW01 已清一批 legacy；仍残留 `/v1/spaces*` · `/v1/feedback`  
> 原则：契约只保留 `/api/v1/*`；映射说明留在 alignment 文档。

| # | Sprint | 状态 |
|---|--------|------|
| **EW101** | 删除 OpenAPI 中 legacy `/v1/*` 路径 | ✅ |
| **EW102** | 签字 | ✅ |

## EW101

从 `openapi-ash-v1.yaml` 去掉无 handler 的 `/v1/spaces`、`/v1/spaces/{spaceId}/members`、`/v1/feedback` 及仅被其引用的 Legacy schema。`make openapi-check`。

## EW102

CHANGELOG + 清单。
