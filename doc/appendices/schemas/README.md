# JSON Schemas（v0.1）

本目录存放可被运行时直接加载的 JSON Schema，用于：
- 事件 payload 校验（写入 `run_events` 前）
- artifacts `manifest.json` 校验
- run summary（`run.json`）校验

文件清单：
- `event-payloads-tr0.v0.1.schema.json`：TR0 必需事件的 payload `$defs`
- `artifact-manifest.v0.1.schema.json`：Artifacts `manifest.json`
- `run-summary.v0.1.schema.json`：RunSummary（`run.json`）
- `internal/rules/schemas/ash.rules.v0.1.schema.json`：Rules DSL（`ParseAndValidate` 运行时 embed 校验）
- `internal/observability/config/schemas/ash.obs.v0.1.schema.json`：Observability 插件配置（Worker 启动 `obsconfig.Load`）
- `internal/events/schemas/event-payloads-tr0.v0.1.schema.json`：TR0 事件 payload（`ASH_VALIDATE_EVENT_PAYLOADS=1` 时 `events.Append` 校验）

