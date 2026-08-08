# 附录 H：Proto（gRPC/Buf）服务定义草案（插件 ABI）v0.1

> 目的：为“进程外插件”提供强约束 ABI（跨平台/跨语言），并可用 buf 做 breaking check。
>
> 冻结等级：M0 提供草案与目录结构；P1 开始落地实现并纳入 TR2-04（隔离）与可观测性输出。

## 1. 目录结构建议
```text
proto/
  ash/
    v1/
      common.proto
      events.proto
      plugin_registry.proto
      observability_exporter.proto
      rag_indexer.proto
  buf.yaml
  buf.gen.yaml
```

## 2. 设计原则
- **版本治理**：`ash.v1` 包名固定；破坏性变更走 v2。
- **强约束**：所有请求/响应必须带 `trace_id/run_id`（如适用）。
- **隔离**：插件失败不得影响主进程；必须超时/熔断；输出不得污染事件流。

## 3. 核心消息（草案）
### 3.1 `common.proto`
- `TraceContext { string trace_id; string run_id; }`
- `Status { string code; string message; google.protobuf.Any details; }`

### 3.2 `events.proto`
- `EventEnvelope { string id; string trace_id; string run_id; int64 seq; int64 ts; string type; bytes payload_json; }`
  - 说明：payload 以 JSON bytes 透传，避免 proto 频繁改动；同时主进程仍以 JSON Schema 校验为准

**已决策（MVP）**：TR0 事件保持 JSON payload 透传 + JSON Schema 校验；不强类型 proto（降低 churn）。需要时再开 v1 事件强类型草案。

## 4. 服务定义（草案）
### 4.1 PluginRegistry（插件注册与健康）
`service PluginRegistry`
- `Register(PluginInfo) returns (RegisterReply)`
- `Heartbeat(TraceContext) returns (HeartbeatReply)`
- `GetStatus(TraceContext) returns (PluginStatus)`

`PluginInfo` 建议包含：`id/version/capabilities/endpoints/supported_events`.

### 4.2 EventStream（事件订阅）
`service EventStream`
- `Subscribe(SubscribeRequest) returns (stream EventEnvelope)`
  - 支持按 `run_id` 或按事件类型过滤

### 4.3 ObservabilityExporter（外部观测输出插件）
`service ObservabilityExporter`
- `ExportMetrics(ExportMetricsRequest) returns (ExportReply)`
- `ExportSpans(ExportSpansRequest) returns (ExportReply)`
- `ExportLogs(ExportLogsRequest) returns (ExportReply)`

### 4.4 RagIndexer（索引/检索插件，P1+）
`service RagIndexer`
- `IndexRepo(IndexRepoRequest) returns (IndexRepoReply)`
- `Query(QueryRequest) returns (QueryReply)`

## 5. Buf 集成建议
- `buf lint`：保证风格一致
- `buf breaking`：保证版本兼容
- `buf generate`：生成 Go/TS（可选）client/server stub

**已决策（MVP）**：Go 生成物进仓（`make proto-generate` / `proto-check`）；TS 客户端可选、非阻断。

## 6. 打包与签名策略（已实现骨架）

| 项 | 约定 |
|----|------|
| 算法 | `hmac-sha256`（`internal/pluginabi/sign.go`） |
| 材料 | `name\\nversion\\nprotocol\\nabi\\nendpoint`（见 `SignMaterial`） |
| 密钥 | `ASH_PLUGIN_SIGNING_KEY`；设为非空则注册必须带有效签名 |
| 强制 | `ASH_PLUGIN_SIGNING_REQUIRED=1`（无密钥视为配置错误） |
| HTTP | `POST /api/v1/plugins` 字段 `signature`（hex） |
| gRPC | `RegisterRequest.signature`（field 10）或 capability `ash.sign.hmac=<hex>` |
| CLI | `ash plugin-sign --name … --version … --endpoint …` |
| 打包建议 | `plugin.json` + `signature.txt` + 二进制/配置 |
| 生产暴露 | `ASH_PLUGIN_GRPC_ADDR` 默认仅在 `ASH_AUTH_MODE=dev` 开启本机监听；生产显式配置并配合签名 |
| 轮换 SOP | [`plugin-signing-sop.md`](../checklists/plugin-signing-sop.md) |
| 烟测 | `make plugin-sign-smoke` |

运行时：`GET /api/v1/plugins/abi` 返回 `signingAlg` / `signingRequired` / `signingKeyConfigured`。

