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

**TODO（负责人：后端）**：评估是否需要为 TR0 事件做强类型 proto（可选）。  
**验收方式**：buf breaking 检查可通过；插件可消费事件流。

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

**TODO（负责人：平台）**：确定生成目标（Go 必需；TS 可选用于前端工具）。  
**验收方式**：CI 中强制 lint+breaking；生成物可编译。

