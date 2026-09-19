# Sprint DX77 — ABI 姿态 / Listen posture on ABI

> **前置 / Prerequisite：** DX76  
> **状态 / Status：** ✅  
> **做法 / Approach：** `GET /api/v1/plugins/abi` 增加 `ragIndexerEnabled` / `ragIndexerGrpcAddr`（与现有 `grpcEnabled` 对称）。不新增页面。  
> Extend `GET /api/v1/plugins/abi` with `ragIndexerEnabled` / `ragIndexerGrpcAddr`, mirroring `grpcEnabled`. No new page.

## 任务板 / Board

| ID | 任务 / Task | 状态 |
|----|-------------|------|
| DX77-1 | 响应字段 + OpenAPI / swagger | ✅ |
| DX77-2 | 单测：默认关 / 配置开；含 `rag_indexer.proto` | ✅ |
| DX77-3 | FE 类型可选字段（无 UI） | ✅ |

## 验收 / Verify

```bash
go test ./internal/api/ -count=1 -run 'PluginABIProfile'
go test ./internal/openapicheck -count=1 -run PluginABI
```
