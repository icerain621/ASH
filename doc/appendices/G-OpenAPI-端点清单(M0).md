# 附录 G：OpenAPI（Swagger）端点清单（M0）v0.1

> 目的：给 Gin + Swagger（`swaggo/swag`）生成与评审提供“端点边界”。M0 只覆盖必需端点，避免范围失控。
>
> 与手写契约、swag 输出的对齐策略见 `doc/api/openapi-alignment.md`；CI 本地校验：`make openapi-check`。
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

**错误码表**：`doc/api/error-codes.md` + `internal/apicodes/catalog.go`（`go test ./internal/apicodes` 与 handler 同步校验）。  
**验收方式**：TR0/1 失败路径返回 `{"error":{"code","message"}}` 统一形状。

## 2. Health / Docs / Metrics
- `GET /healthz`：存活探针 → `HealthResponse`
- `GET /readyz`：就绪探针（DB/关键依赖检查）→ `HealthResponse`（含 `dialect`/`schemaMode`/`otelEnabled`/`alertsEvalInterval`/`metricsEventReplayEnabled`）
- `GET /metrics`：Prometheus 指标（全局运维 scrape；RLS 开启时 bypass）
- `GET /api/v1/metrics/prometheus`：租户范围 Prometheus 文本（需 `observability:read`）
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
  - body：`{ suite: "TR0"|"TR1"|"TR2"|"TR3"|"ALL", format: "json"|"md" }`
  - resp：`{ reportId }`
- `GET /doctor/reports/{reportId}`
  - resp：报告内容（json/md）

## 8. M1 / TR2 / TR3（已实现，需 `make swagger` 同步 OpenAPI）

### 8.1 自我迭代（M1）
- `POST /improve/proposals`、`GET /improve/proposals`、`GET /improve/proposals/{id}`
- `POST .../experiment`、`POST .../canary`、`POST .../promote`、`POST .../rollback`

### 8.2 合规（TR2）
- `GET /compliance/secret-scan` — 审计/事件载荷 secret 模式扫描
- `POST /compliance/export` — 审计包（含 Doctor 报告 + `secretScan` 摘要）
- `GET /data-policy` — 数据分级与保留期生效快照（附录 J）
- `POST /events/retention/apply` — 按空间清理过期 `run_events`
- `POST /artifacts/retention/apply` — 按空间清理过期 `artifact_index`（可选删对象文件）

### 8.3 规模化（TR3）
- `GET /scale/readiness` — `ScaleReadinessResponse`（见 §8.6）
- `GET /runs/{runId}/provenance` — `ProvenanceResponse`（`runId`/`traceId`/`scenario`/计数/`links[]`）

### 8.3.1 合规（TR2）OpenAPI schema
- `GET /compliance/secret-scan` — `SecretScanResponse`（`findings[]`/`leakCount`/`redactEnabled`）
- `POST /compliance/export` — `ComplianceExportResponse`（202；含 `DoctorReportResponse`）

### 8.3.2 可观测 / RAG / 插件
- `GET /rag/profile` — `RAGProfileResponse`
- `GET /observability/otel/status` — `OtelStatusResponse`
- `GET /plugins/health` — `PluginHealthSummary`

### 8.3.3 Runs / Doctor
- `GET /runs` — `RunListResponse`（`items[]` → `RunSummaryResponse`）
- `POST /runs` — `RunCreateResponse`（201）
- `GET /runs/{runId}` — `RunSummaryResponse`
- `POST /doctor/run` — `DoctorRunRequest` → `DoctorRunResponse`
- `GET /doctor/reports/{reportId}` — `DoctorReportResponse`

### 8.3.4 Runs 检查 / 产物 / 场景
- `GET /runs/{runId}/tool-calls` — `ToolCallListResponse`
- `GET /runs/{runId}/agent-tasks` — `AgentTaskListResponse`
- `GET /runs/{runId}/quality-metrics` — `QualityMetricListResponse`
- `GET /runs/{runId}/artifacts` — `ArtifactsManifestResponse`
- `GET /runs/{runId}/artifacts/{name}/access` — `ArtifactAccessResponse`
- `GET /runs/{runId}/timeline` — `TimelineAPIResponse`
- `GET /scenarios` — `ScenarioListResponse`
- `GET /scenarios/{name}/{version}` — `ScenarioDetailResponse`
- `POST /scenarios/validate` — `ValidateScenarioRequest` → `ValidationResponse`
- `GET /storage/profile` — `StorageProfileResponse`

### 8.3.5 KPI
- `GET /metrics/overview` — `MetricsOverviewResponse`

### 8.4 平台（补充）
- `GET /spaces/{spaceId}/resource-scopes` — 资源作用域列表
- `PUT /spaces/{spaceId}/resource-scopes/{scopeId}` — 更新场景工具策略（`policyJson`）

### 8.5 M2 权限矩阵
- `GET /permissions/matrix` — `PermissionMatrixResponse`（`catalog`/`builtinRoles`/`scenarioTools`）
- `GET /spaces/{spaceId}/permissions/matrix` — 同上
- 创建空间时自动种子 `scenario` 资源作用域（三场景 × 角色工具 allow/deny）
- 创建 Run 时写入 `actorRole`；工具链执行前按场景矩阵校验并发出 `policy.denied`
- 更新场景策略写入 `scope.policy_updated` 审计

### 8.6 M3 规模化 / 数据库
- `GET /scale/readiness` 返回 `ScaleReadinessResponse`（手写契约 `doc/api/openapi-ash-v1.yaml` 与 swag 字段对齐）：
  - **租户数据**：`spaceId`、`memoryApprovedCount`、`ragDocumentCount`、`ragChunkCount`、`qualityMetricRows`、`auditLogRows`
  - **数据库 / 迁移**：`databaseDialect`、`postgresConfigured`、`migrationReady`、`migrationTableCount`、`sqlitePath`、`dualWrite*`、`lastMigrationSync*`
  - **Postgres RLS / 角色**：`postgresRLSEnabled`、`postgresRLSForce`、`postgresRLSPolicyCount`、`postgresAppUrlConfigured`、`workerConnectionRole`、`runtimeDsnHint`
  - **Schema 模式**：`schemaMode`、`sqlMigrationsEnabled`、`autoMigrateEnabled`、`sqlMigrationVersion`、`sqlMigrationExpected`、`readinessWarnings`
  - **记忆 / RAG**：`memorySchemaVersion`、`memoryCatalogVersion`、`memoryPendingMigrationRecords`、`ragFtsAvailable`、`ragFtsEngine`、`ragDefaultRetrievalMode`、`ragFallbackQueryCount`
  - **可观测 / 运维**：`modelUsageRows`、`modelCostMicrosTotal`、`otelEnabled`、`alertsEvalInterval`、`metricsEventReplayEnabled`

## 9. OpenAPI 契约与规划路径（M0+）

- **手写契约**：`doc/api/openapi-ash-v1.yaml`（产品承诺 + M0 规划）
- **实现真源**：`internal/api/docs/swagger.yaml`（`make swagger` 再生）
- **CI 校验**：`make openapi-check`（path 子集 + schema 字段对齐 + swag 确定性）
- **Doctor TR3-09**：`openapicheck.ValidateContract`（`/api/v1` 路径覆盖 + 2xx 无泛型 `ApiResponse`）

### 9.1 已实现 Base Path（`/api/v1/*`）

全部 2xx JSON 成功响应绑定具体 `components.schemas`（见 `openapi-alignment.md`）。核心域包括：Runs/Doctor、Memory、Compliance/Scale、CI/KPI、Observability/RAG、Org/Auth、Releases/Approvals/Audit、Improve/Plugins。

### 9.2 M0 规划路径（`/v1/*`，未实现）

| 规划 | 实现对照 | 响应 schema |
|------|----------|-------------|
| `POST /v1/tasks` | `/api/v1/runs` + 编排 | `CreateTaskResponseData` |
| `GET /v1/tasks/{taskId}` | 规划中 | `TaskDetailResponse` |
| `POST /v1/memories` | `POST /api/v1/memory/candidates` | `LegacyMemoryCreateResponse` |
| `POST /v1/memories/search` | `POST /api/v1/memory/query` | `MemoryQueryResponse` |
| `GET /v1/agent-runs/{runId}` | `GET /api/v1/runs/{runId}` | `AgentRun` |
| `GET /v1/runs/{runId}/stream` | `GET /api/v1/runs/{runId}/stream` | `text/event-stream` |
| `POST /v1/spaces` | `POST /api/v1/spaces` | `SpaceItem` |
| `POST /v1/spaces/{spaceId}/members` | `POST /api/v1/spaces/{spaceId}/members` | `MemberItem` |
| `POST /v1/feedback` | `POST /api/v1/feedback` | `FeedbackItem` |

