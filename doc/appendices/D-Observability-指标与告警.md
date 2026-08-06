# 附录 D：Observability（插件化）— 指标与告警（v0.1）

> 本附录定义 ASH 的可观测性实现要求：插件配置（ash.obs/v0.1）、关键指标清单、事件→指标/Span 派生规则与告警建议。
>
> 冻结等级：**M0 冻结**（事件为真相 + Prometheus 基础指标 + redaction/sampling）。OTel/高级插件为 P1+。

## 1. 插件化总览
- **真相来源**：`run_events`
- **派生输出**：Prometheus / OTel / Console / sqlite-only 等插件
- **安全**：redaction 默认开启；外发插件默认关闭（组织策略决定）

## 2. 插件配置（ash.obs/v0.1）
**位置建议**：`config/ash-observability.yaml`（或 `~/.ash/observability.yaml`）。

**JSON Schema**：`internal/observability/config/schemas/ash.obs.v0.1.schema.json`；默认文件 `config/ash-observability.yaml`；Worker 启动时 `obsconfig.Load()` 校验。  
**验收方式**：`go test ./internal/observability/config -run ValidateSchema`（含 10+ 非法样例）；`allowOutbound=true` 须 `redaction.enabled=true`。

## 3. 核心指标清单（M0 必须）
> 命名建议采用 Prometheus 风格。

### 3.1 Run / Step
- `ash_run_total{scenario,status}`（counter）
- `ash_run_duration_ms{scenario}`（histogram）
- `ash_run_inflight{scenario}`（gauge）
- `ash_step_total{scenario,stepId,status}`（counter）
- `ash_step_duration_ms{scenario,stepId,role}`（histogram）
- `ash_checkpoint_saved_total{scenario}`（counter）
- `ash_resume_total{scenario}`（counter）

### 3.2 Tool / Policy
- `ash_tool_calls_total{tool,risk,ok}`（counter）
- `ash_tool_duration_ms{tool}`（histogram）
- `ash_policy_denied_total{target,reason}`（counter）

### 3.3 Model / Cost
- `ash_model_requests_total{provider,model,ok}`（counter）
- `ash_model_latency_ms{provider,model}`（histogram）
- `ash_token_in_total{provider,model}`（counter）
- `ash_token_out_total{provider,model}`（counter）
- `ash_cost_estimated_total{currency}`（counter）

### 3.4 RAG / Citations
- `ash_rag_queries_total{source}`（counter）
- `ash_rag_latency_ms{source}`（histogram）
- `ash_rag_citation_missing_total{scenario,stepId}`（counter）

### 3.5 Memory（记忆可观测性）
- `ash_memory_candidates_total{layer}`（counter）
- `ash_memory_reviews_total{layer,decision}`（counter）
- `ash_memory_review_latency_ms{layer}`（histogram）
- `ash_memory_unreviewed_backlog{layer}`（gauge）
- `ash_memory_missing_evidence_total{layer}`（counter）
- `ash_memory_queries_total{layersKey}`（counter）
- `ash_memory_query_latency_ms`（histogram）
- `ash_memory_hit_used_total{layer}`（counter）
- `ash_memory_deprecated_total{layer,reason}`（counter）
- `ash_memory_migration_runs_total{from,to,ok}`（counter）

## 4. 事件 → 指标派生（摘要规则）
> 指标从事件派生，事件可重算；监控系统的数据允许丢失但事件不可丢。

- `run.started` → `ash_run_total{status="started"}++`、`ash_run_inflight++`
- `run.finished` → `ash_run_total{status="finished"}++`、`ash_run_inflight--`、`ash_run_duration_ms.observe`
- `run.failed` → `ash_run_total{status="failed"}++`、`ash_run_inflight--`
- `run.canceled` → `ash_run_total{status="canceled"}++`、`ash_run_inflight--`
- `step.finished` → `ash_step_total++`、`ash_step_duration_ms.observe`
- `tool.result` → `ash_tool_calls_total{ok}++`、`ash_tool_duration_ms.observe`
- `policy.denied` → `ash_policy_denied_total++`
- `model.usage` → tokens/cost 相关 counter 增量
- `rag.results` → `ash_rag_latency_ms.observe`、`ash_rag_citation_missing_total`（若缺引用）
- `rag.retrieved` → `ash_rag_queries_total{mode}`、`ash_rag_fts_fallback_total`（`retrievalMode=chunk` 时）
- **RAG FTS 引擎**：SQLite `fts5` 虚拟表；Postgres `rag_chunks.search_vector`（`tsvector` + GIN，SQL rev **17**）
- `memory.candidate_created` → candidates/missing_evidence/backlog
- `memory.reviewed` → reviews/latency/backlog
- `memory.hit_used` → `ash_memory_hit_used_total{layer}`（按 `hitsByLayer` 分层计数）
- `memory.deprecated` → `ash_memory_deprecated_total{layer,reason}`
- `memory.query` → `ash_memory_queries_total{layersKey}`、`ash_memory_query_latency_ms`（需 `runId` 写入 run_events）
- `memory.migrated` → `ash_memory_migration_runs_total{from,to,ok}`

**状态**：TR0 核心 + 记忆 P1 事件已表驱动实现（`internal/observability/derive/catalog.go`）。  
**验收方式**：对同一 run 的事件重放，离线重算指标与实时指标口径一致（Doctor **TR3-05**、`derive.ValidateReplayParity`）。

## 5. Traces（OTel 骨架）
建议 Span 树：
- `run`（root）
  - `step:*`
    - `gate:*`
    - `rag.query`
    - `model.chat`
    - `tool.call`

关键 attributes：`runId/scenarioVersion/stepId/role/tool/provider/model/checkpointId`。

**实现**：`internal/observability/otel`（Provider、Span 辅助、waterfall 批量导出）。  
**配置**：`config/ash-observability.yaml` → `plugins.otel`；或 `ASH_OTEL_ENABLED=1` + `ASH_OTEL_ENDPOINT`。  
**运行时**：Worker 启动 `otel.Init`；Run 执行 live span；Run 结束后 `ExportWaterfall` 将 DB waterfall 同步为 OTLP batch。  
**API**：`GET /api/v1/observability/otel/status`。  
**验收**：`go test ./internal/observability/otel/...`；启用 OTel 后 run 完成可在 collector 看到 span 树。

## 6. 告警建议（M0）
### 6.1 可用性
- Run 失败率在 10min 窗口超过阈值（按 scenario）
- SSE 续传失败率上升（合成探测）

### 6.2 安全
- `ash_policy_denied_total` 突增（可能注入攻击或规则过严）
- redaction 扫描发现疑似 secret（必须 P0 阻断外发）

### 6.3 记忆治理
- `ash_memory_unreviewed_backlog{layer=L1/L2}` 持续增长
- `ash_memory_missing_evidence_total` 比例超过阈值
- 评审 SLA 超时（P1 细化）
- **默认告警规则**（`internal/alerts`）：`memory_unreviewed_backlog`、`rag_fts_fallback_rate`、`plugin_export_failures`、`run_inflight_count`
- **Prometheus live 段**：`ash_memory_unreviewed_backlog_live`、`ash_rag_fts_fallback_live`、`ash_plugin_export_errors_live`、`ash_run_inflight_live`

## 7. 插件健康与自监控
- `plugin_registry` 列：`last_export_at`、`export_errors`、`drop_count`（SQL revision **16**）
- 插件侧或 Worker 通过 `POST /api/v1/plugins/{pluginId}/export-report` 上报 `{ ok?, dropped? }`
- 运维快照：`GET /api/v1/plugins/health`（汇总 export 错误、丢弃计数、24h 未导出插件数）
- UI：**Automation → Plugins** 表格展示每插件导出健康列
- 插件导出失败不影响 run 主流程，但必须可观测
- `plugin.export_failed` / `plugin.export_reported` 写入 `audit_log`；可选 `runId` 关联 run 事件 → `ash_plugin_export_failures_total`

