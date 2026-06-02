# 附录 G：OpenAPI（Swagger）端点清单（M0）v0.1

> 目的：给 Gin + Swagger（`swaggo/swag`）生成与评审提供“端点边界”。M0 只覆盖必需端点，避免范围失控。
>
> 冻结等级：**M0 冻结**（v0.1）。字段新增需向后兼容；破坏性变更必须升 API 版本或保持旧字段。

## 1. 通用约定
- **Base Path**：`/api/v1`
- **Auth（M0 最小）**：可先支持 `Bearer Token`（可选），或内网模式下禁用；生产建议强制启用并接入 RBAC/ABAC（P2）。
- **Trace/Run 关联**：所有响应建议带 `X-Trace-Id`、`X-Run-Id`（如适用）。
- **分页**：`limit`/`cursor`（或 `offset`/`limit`，建议 cursor）。
- **错误形状（统一）**
  - `code`：稳定错误码
  - `message`：可读信息
  - `details`：可选扩展

**TODO（负责人：后端）**：补齐错误码表（tool/model/rag/memory/rules）。  
**验收方式**：TR0/1 所有失败路径均返回统一错误形状。

## 2. Health / Docs / Metrics
- `GET /healthz`：存活探针
- `GET /readyz`：就绪探针（DB/关键依赖检查）
- `GET /metrics`：Prometheus 指标
- `GET /docs`：Swagger UI（或 `/swagger/*any`）
- `GET /openapi.json`：OpenAPI JSON（swag 生成）

## 3. Runs（运行与回放）
### 3.1 创建/查询 Run
- `POST /runs`
  - body：`{ scenario: {name, scenarioVersion}, inputs, policyProfile? }`
  - resp：`{ runId, traceId }`
- `GET /runs`
  - query：`scenarioName? status? from? to? repo?`
  - resp：`{ items: [RunSummary], nextCursor? }`
- `GET /runs/{runId}`
  - resp：`RunSummary`

### 3.2 控制 Run
- `POST /runs/{runId}/resume`
- `POST /runs/{runId}/pause`（可选，M0 可先不实现但预留）
- `POST /runs/{runId}/replay`
  - body：`{ mode: "exact"|"latest_memory", overrides? }`
  - resp：`{ runId: newRunId }`

### 3.3 事件流（SSE）
- `GET /runs/{runId}/stream`
  - headers：`Last-Event-ID`（或 `Last-Event-Seq`）
  - resp：SSE（`EventEnvelope`）

### 3.4 Artifacts（交付物）
- `GET /runs/{runId}/artifacts`
  - resp：`{ manifest, artifacts: [...] }`
- `GET /runs/{runId}/artifacts/{artifactId}/download`
  - resp：文件流

## 4. Rules / Scenarios（DSL 管理）
- `GET /scenarios`
  - resp：可用模板列表（name, scenarioVersion, description）
- `GET /scenarios/{name}/{scenarioVersion}`
  - resp：DSL 原文（YAML）+ 校验结果
- `POST /scenarios/validate`
  - body：DSL（YAML 文本）
  - resp：`{ ok, errors[] }`

## 5. Memory（记忆治理）
### 5.1 候选与评审
- `POST /memory/candidates`
  - body：`MemoryCandidate`（含 evidence，L1+ 必须）
  - resp：`{ candidateId }`
- `GET /memory/candidates`
  - query：`layer? status=candidate repo?`
  - resp：分页列表
- `POST /memory/candidates/{candidateId}/review`
  - body：`{ decision: approve|reject|deprecate, reason, policyProfile }`
  - resp：`{ ok: true }`

### 5.2 查询与使用记录
- `POST /memory/query`
  - body：`{ text, layers[], scope?, topK? }`
  - resp：`{ items: [MemoryRecord] }`
- `POST /memory/hit-used`
  - body：`{ runId, recordIds[] }`
  - resp：`{ ok: true }`

## 6. Observability（可选管理面）
- `GET /observability/plugins`
  - resp：插件启用状态与健康（脱敏）
- `POST /observability/plugins/reload`
  - resp：`{ ok: true }`

## 7. Doctor（验证）
- `POST /doctor/run`
  - body：`{ suite: "TR0"|"TR1"|"TR2"|"ALL", format: "json"|"md" }`
  - resp：`{ reportId }`
- `GET /doctor/reports/{reportId}`
  - resp：报告内容（json/md）

