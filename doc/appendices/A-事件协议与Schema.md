# 附录 A：事件协议与 Schema（v0.1）

> 本附录定义 ASH 的统一事件协议（EventEnvelope）、事件类型表、SSE 续传规则、事件持久化表结构与回放约束。
>
> 冻结等级：**M0 冻结**（v0.1）。新增字段需向后兼容；废弃字段需保留过渡期。

## 1. 术语
- **Run**：一次可回放的执行单元（绑定 scenarioVersion、输入、工具调用、产物、记忆版本）。
- **TraceId**：一次 run 的链路标识（同 run 内一致）。
- **Event**：运行过程中产生的事实记录（可落盘、可续传、可回放、可派生指标）。

## 2. EventEnvelope（统一包络）
### 2.1 结构（JSON）
```json
{
  "id": "evt_00000123",
  "traceId": "trc_xxx",
  "runId": "run_xxx",
  "seq": 123,
  "ts": 1714310000000,
  "type": "tool.called",
  "severity": "info",
  "payload": {}
}
```

### 2.2 字段规范
- **id**：全局唯一（建议 ULID/UUID），用于审计与跨系统关联。
- **runId**：Run 唯一标识。
- **seq**：run 内单调递增序列（SSE 续传与回放的有序依据）。
- **type**：事件类型（见第 4 节）。
- **payload**：事件负载（需通过类型对应的 payload schema 校验）。

本 v0.1 在 M0 阶段至少冻结 TR0 必需事件的 payload schema（见第 7 节）。

## 3. SSE 端点与续传
### 3.1 端点
- `GET /runs/:runId/stream`（SSE）

### 3.2 续传机制
- 客户端断线重连带 `Last-Event-ID`（建议携带 `evt_...` 或 `seq`）。
- 服务端根据 `run_events` 表按 `seq > lastSeq` 回放补齐，再进入实时推送。
- **要求**：run 内事件不重不漏（按 `seq`）；允许跨连接重复推送时客户端按 `seq` 去重。

## 4. 事件类型表（最小全集）
### 4.1 Run 生命周期
- `run.started`
- `run.checkpoint_saved`
- `run.paused`
- `run.resumed`
- `run.finished`
- `run.failed`

### 4.2 Step 生命周期
- `step.started`
- `step.progress`
- `step.finished`

### 4.3 Model
- `model.selected`
- `model.chunk`
- `model.usage`

### 4.4 RAG / Citations
- `rag.query`
- `rag.results`

### 4.5 Tool / Policy
- `tool.called`
- `tool.result`
- `policy.denied`

### 4.6 Memory
- `memory.candidate_created`
- `memory.review_requested`
- `memory.reviewed`
- `memory.hit_used`
- `memory.ttl_expired`
- `memory.deprecated`
- `memory.migrated`

### 4.7 Improvement（P1+）
- `improve.proposal_created`
- `improve.experiment_started`
- `improve.experiment_finished`
- `improve.canary_started`
- `improve.rollback`
- `improve.promoted`

## 5. 事件持久化（run_events）
### 5.1 表结构（SQLite/PG）
```sql
CREATE TABLE IF NOT EXISTS run_events (
  id TEXT PRIMARY KEY,
  run_id TEXT NOT NULL,
  seq INTEGER NOT NULL,
  ts INTEGER NOT NULL,
  type TEXT NOT NULL,
  payload_json TEXT NOT NULL,
  created_at INTEGER NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_run_seq ON run_events(run_id, seq);
CREATE INDEX IF NOT EXISTS idx_run_type ON run_events(run_id, type);
```

### 5.2 写入规则
- 每写入一个事件必须分配新的 `seq`（run 内单调递增）。
- 写入应为**异步/缓冲**，避免阻塞主执行路径；失败必须 emit `run.failed` 或 `observability.export_error`（可选）。

## 6. 回放约束（Replay）
回放（Replay）需要三类锚点：
- **事件序列**：`run_events`（按 seq）
- **产物完整性**：artifacts digest（diff/test_report 等）
- **记忆一致性**：run 引用的 memory record ids + schemaVersion

Artifact digest 与存储规范见 `docs/appendices/F-Artifacts规范与Digest.md`。  
**验收方式**：同一 run 回放 10 次，四件套 artifacts digest 一致率 100%（允许时间戳差异）。

## 7. TR0 必需事件 payload schema（v0.1 冻结）
> 说明：这里给出“字段级 schema”（便于评审）。运行时可直接加载 `docs/appendices/schemas/event-payloads-tr0.v0.1.schema.json` 中的 `$defs` 进行校验，并在写入 `run_events` 前执行。

### 7.1 `run.started`
- **payload**
  - `scenario`: `{ name: string, scenarioVersion: string }`
  - `policyProfile`: string
  - `inputsDigest`: string（对输入的摘要，用于回放对照）
  - `repo`: `{ root: string, revision?: string, branch?: string }`（可选）

### 7.2 `step.started`
- **payload**
  - `stepId`: string
  - `role`: string
  - `kind`: `"llm" | "tool_chain" | "human"`

### 7.3 `step.finished`
- **payload**
  - `stepId`: string
  - `ok`: boolean
  - `durationMs`: number
  - `artifacts`: `Array<{ id: string, type: string, digest?: string }>`（可选）
  - `memoryCandidates`: `Array<{ layer: string, title: string }>`（可选）
  - `error?`: `{ code: string, message: string }`

### 7.4 `run.checkpoint_saved`
- **payload**
  - `checkpointId`: string
  - `stepId`: string
  - `snapshotDigest`: string
  - `strategy`: `"per_step" | "per_tool" | "manual"`

### 7.5 `tool.called`
- **payload**
  - `tool`: string
  - `risk`: `"safe" | "restricted" | "danger"`
  - `timeoutMs`: number
  - `argsDigest`: string

### 7.6 `tool.result`
- **payload**
  - `tool`: string
  - `ok`: boolean
  - `durationMs`: number
  - `outputDigest?`: string
  - `artifacts?`: `Array<{ id: string, type: string, digest?: string }>`
  - `error?`: `{ code: string, message: string, details?: unknown }`

### 7.7 `model.selected`
- **payload**
  - `providerId`: string
  - `modelId`: string
  - `caps`: `{ streaming: boolean, toolCalling: boolean, jsonSchema: boolean, maxContextTokens?: number }`
  - `budget`: `{ maxTokens?: number, maxCost?: number, currency?: string }`（可选）

### 7.8 `model.usage`
- **payload**
  - `providerId`: string
  - `modelId`: string
  - `inTokens`: number
  - `outTokens`: number
  - `costEstimated?`: number
  - `currency?`: string

### 7.9 `rag.query`
- **payload**
  - `queryDigest`: string（避免泄露原文）
  - `sources`: `Array<"code"|"docs"|"memory"|"artifacts">`
  - `topK`: number
  - `requireCitations`: boolean

### 7.10 `rag.results`
- **payload**
  - `durationMs`: number
  - `results`: `Array<{ ref: string, digest?: string, score?: number }>`（只允许摘要/引用，不允许塞全文）
  - `citationsMissing?`: boolean

### 7.11 `policy.denied`
- **payload**
  - `target`: `"tool" | "model" | "memory" | "gate"`
  - `reason`: string
  - `action`: `"deny" | "degrade" | "require_human"`
  - `ref?`: string（例如 tool 名或 gateId）

### 7.12 `memory.candidate_created`
- **payload**
  - `candidateId`: string
  - `layer`: `"L0"|"L1"|"L2"|"L3"`
  - `evidenceCount`: number
  - `sensitivity`: `"normal"|"restricted"|"secret"`

### 7.13 `memory.reviewed`
- **payload**
  - `candidateId`: string
  - `decision`: `"approve"|"reject"|"deprecate"`
  - `reason`: string
  - `policyProfile`: string
  - `latencyMs?`: number（candidate→reviewed）

### 7.14 `run.finished`
- **payload**
  - `ok`: true
  - `durationMs`: number
  - `artifacts`: `Array<{ id: string, type: string, digest: string }>`（四件套必须在此出现）
  - `metrics?`: `{ toolFailures?: number, citationMissing?: number, recovered?: boolean }`（可选）

### 7.15 `run.failed`
- **payload**
  - `ok`: false
  - `durationMs`: number
  - `error`: `{ code: string, message: string, recoverable?: boolean }`
  - `lastCheckpointId?`: string

