# Sprint DX73 — 生产插件监听 / Production plugin listen

> **前置 / Prerequisite：** v4.1 已冻结；[`v4.2-release-scope.md`](v4.2-release-scope.md)  
> **状态 / Status：** ✅  
> **做法 / Approach：** 非 dev 且 `ASH_PLUGIN_GRPC_ADDR` 非空时，缺密钥或未 `ASH_PLUGIN_SIGNING_REQUIRED=1` 则拒绝启动。dev 与空地址保持原样。  
> Non-dev with a plugin gRPC address refuses to start unless the signing key is set and signing is forced. Dev mode and an empty address stay unchanged.

## 任务板 / Board

| ID | 任务 / Task | 状态 |
|----|-------------|------|
| DX73-1 | `ProductionPluginGRPCListen` | ✅ |
| DX73-2 | worker 启动前调用 | ✅ |
| DX73-3 | 单测：dev 放行 / 生产缺料拒绝 / 双料放行 | ✅ |

## 验收 / Verify

```bash
go test ./internal/pluginabi/ -count=1 -run TestProductionPluginGRPCListen
```
