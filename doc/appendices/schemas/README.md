# JSON Schemas（v0.1）

本目录存放可被运行时直接加载的 JSON Schema，用于：
- 事件 payload 校验（写入 `run_events` 前）
- artifacts `manifest.json` 校验
- run summary（`run.json`）校验

文件清单：
- `event-payloads-tr0.v0.1.schema.json`：TR0 必需事件的 payload `$defs`
- `artifact-manifest.v0.1.schema.json`：Artifacts `manifest.json`
- `run-summary.v0.1.schema.json`：RunSummary（`run.json`）

