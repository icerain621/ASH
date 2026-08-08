# ASH 架构与技术选型 v0.1

> 目的：解释“为何这样选”，明确默认实现、可替换项、演进路线与风险。
>
> 文档状态：v0.2（对照 `v0.1.0-mvp` 实现修订，2026-08-08）。未决项以 **TODO** 标注；已落地项标注「已实现」。
> 进度真相源：[`../plan/PLAN-进度与里程碑.md`](../plan/PLAN-进度与里程碑.md)。  
> 归属：[`design/`](README.md)

## 1. 选型原则

- **可替换**：provider/RAG/向量库/MCP/观测插件可插拔。
- **可审计、可回放**：事件与产物为真相来源，外部监控为派生视图。
- **安全优先**：危险工具默认 deny；最小权限；脱敏。
- **渐进演进**：M0 先“可用可交付”，P2 再“强隔离与组织级治理”。

## 2. 总体拓扑

- **应用形态**：Monorepo（文档在 `doc/`，实现在仓库根）
  - `cmd/worker`（Go 编排服务：Gin + GORM + SSE）— **已实现**
  - `cmd/cli`（Go CLI：run/replay/doctor）— **已实现**
  - `frontend/`（Vite + React 控制台，Worker `/ui/`）— **已实现**
  - `doc/appendices/schemas`（JSON Schema 与规范资产）

## 3. 语言与运行时

- **后端默认（优先）**：**Go**（Worker + CLI 同栈）
  - 价值：单文件交付、启动快、部署与资源占用更可控；适合长运行编排服务与本地 CLI；并发模型清晰（goroutine）。
- **前端**：见第 3.3 节建议（React 生态）
- **Python（可选）**：作为外部 tools/索引器（通过 ToolBus 调用），不作为核心运行时依赖

### 3.1 Go 后端技术栈建议（M0 默认）

- **Web 框架**：**Gin**（你指定，生态成熟，适配 Swagger 与中间件体系）
- **ORM**：**GORM**（你指定；支持 SQLite/Postgres；用于 memory/audit/run_events 等持久化）
- **数据库迁移**：建议采用 `golang-migrate` 或 GORM migrator（M0 先简单，P1+ 强化迁移流程）
- **SSE**：基于 `net/http` 直接实现（`ResponseWriter` + flush），事件源来自 `run_events`
- **Swagger / OpenAPI**：建议采用 `swaggo/swag`（Gin 生态成熟）生成 OpenAPI；或以 `openapi.yaml` 作为单一真相（P1+）
- **Proto（gRPC）**：建议采用 `buf` 管理 proto（lint/breaking/check），并提供 gRPC 服务用于插件/内部服务调用（见 3.2）
- **JSON Schema 校验**：运行时加载 `doc/appendices/schemas/*.json`（及 `internal/rules/schemas`），对事件 payload / manifest / run summary 做校验
- **SQLite**：优先 `modernc.org/sqlite`（纯 Go，跨平台友好）或 `mattn/go-sqlite3`（需 CGO）；Windows 场景建议优先纯 Go
- **配置**：`viper` 或自研小型配置加载（env + file + flags）
- **可观测性**：
  - Prometheus：`prometheus/client_golang`
  - OpenTelemetry：`go.opentelemetry.io/otel`（P1+）

### 3.2 插件化集成方式（关键：Windows 友好）

ASH 需要“可选插件”但 **Go 原生 `plugin` 在 Windows 不可用**，因此建议默认采用：

- **进程内插件（内置）**：通过接口注册（编译进二进制），用于 `prometheus/console/sqlite-only` 等内置插件
- **进程外插件（推荐）**：通过 **Proto/gRPC（本机）** 或 **HTTP（本机）** 或 **MCP** 作为扩展点（统一走 ToolBus/Policy）
  - 推荐优先 **Proto/gRPC**：便于版本治理（buf breaking）、跨语言扩展、接口强约束
  - 优点：跨平台、隔离更强、可控权限
- **WASM 插件（P2+）**：用于更强的安全隔离与分发（可选）

**已实现（基础）**：HTTP 插件注册 + 可选 gRPC（`ASH_PLUGIN_GRPC_ADDR`，见 `internal/pluginabi`）。  
**TODO（负责人：平台）**：确定插件**打包/签名**策略与生产暴露边界（P2）。  
**验收方式**：在 Windows 上加载一个签名校验通过的外部观测插件，确保不影响主流程且可审计。

### 3.3 前端技术栈推荐

优先给两个可选默认（你可二选一）：

- **方案 A（推荐）**：**Next.js（React）** + Tailwind（适合“控制台型”产品：路由/SSR/部署生态成熟）
- **方案 B（轻量）**：**Vite + React + TanStack Router/Query**（更轻、更快迭代；适合内网控制台）

共同要求：

- SSE 客户端 + Timeline/Artifacts/Gates/Memory Review 页面（M0）
- 后续可加入 Waterfall（OTel）与质量洞察（P1）

## 4. 编排引擎（WorkflowGraph）

- **思想**：LangGraph 类“状态图/节点/动态路由 + checkpoint”
- **M0 默认实现**：自研轻量状态机（避免引入重依赖），接口与语义对齐 DeerFlow 风格能力：
  - per_step checkpoint
  - 动态路由（按 gate/step 输出 next）
  - 事件驱动（run/step/tool/model/rag）

**替换项**：未来可接入 LangGraph / Temporal / 自研 DAG 引擎（保持接口不变）。

## 5. 存储选型

### 5.1 事件与回放

- **默认**：SQLite（`run_events`）+ SSE 续传
- **演进**：Postgres（P2）用于高并发/多租户

### 5.2 记忆（Memory）

- **默认**：SQLite 元数据（records/evidence/reviews/edges/migrations）
- **向量库**：M0 可不强依赖（先以 FTS/简单检索起步）；P1 引入可插拔向量库：
  - **Chroma**（本地快速）
  - **Qdrant**（服务化）
  - **Milvus**（大规模）

**已决策（M0/MVP）**：不强制向量库；默认 FTS（SQLite FTS5 / Postgres `tsvector`），不可用时 `retrievalMode=chunk|empty` 降级（Doctor TR3-06、derive 指标）。  
**TODO（负责人：平台）**：P3 引入可插拔向量库时的接口与降级 SLO（非 v0.1 范围）。

### 5.3 审计（Audit）

- **M0**：SQLite `audit_log`（索引）+ 文件落盘（可选）
- **P2 建议**：append-only/WORM（对象存储或专用审计库）以满足合规

## 6. RAG 与检索选型

- **M0**：repo scan + 规则化引用输出（不做复杂 AST）
- **P1**：Hybrid 检索（FTS/BM25 + 向量 + rerank）+ 轻量符号索引（ctags/tree-sitter/语言服务之一）

**TODO（负责人：检索）**：确定符号索引技术路线与性能基线（P95）— **优先级降为 P3**（MVP 以 FTS 为准）。  
**验收方式**：以 1 个中型 repo 跑检索基线（延迟/命中/引用准确率）。

## 7. MCP / Skills 集成

- **M0**：ToolBus 内置 native tools；MCP client 接口预留
- **P1**：MCP 工具接入（schema 校验/参数白名单/超时隔离）

## 8. 可观测性选型（插件化）

- **真相来源**：`run_events`
- **M0 默认**：
  - `console` 插件
  - `prometheus` 插件（/metrics）
  - `sqlite-only`（不外发）
- **P1**：
  - `otel` 插件（OTLP exporter）
  - `memory-quality` 插件（治理与告警）
  - `cost-guard`（预算与降级）

**安全默认**：redaction=balanced/strict；外发插件默认关（按组织策略）。

## 9. 安全与隔离路线

- **M0**：策略门禁（policy/hook/gate）+ 最小工具集 + 危险工具 deny + 脱敏
- **P2**：执行隔离（容器/沙箱/受控网络）+ 组织级审批

**已实现（基础）**：M2 policy enforcement、`waiting_approval`、危险工具默认 deny、合规 Secret 扫描与 Redact。  
**TODO（负责人：安全/平台）**：产品化「危险操作列表」清单与控制台 human-step UX 文案（P2 硬化）。  
**验收方式**：TR2 红队用例集全部拦截；审计可追踪。

## 10. 与参考项目的映射（能力对齐）

- **DeerFlow**：状态图编排 + checkpoint 思想
- **Claude‑Mem**：SSE viewer + 本地存储/检索编排思路
- **MemPalace**：分层记忆治理 + MCP 工具面思想
- **Everything‑Claude‑Code / Claude‑Code**：hooks/规则 + 安全基线思路
- **Hermes‑Agent / LangChain**：provider 抽象与生态接入思路
- **OpenClaw（可选）**：网关与 role/scope（P3）

## 11. 是否需要微服务/分布式（结论与原则）

- **M0/M1 结论**：**不建议**做微服务/分布式，优先采用 **单体（Go Worker）+ 插件化扩展（Proto/gRPC/MCP/HTTP）**。
  - 原因：ASH 的“真相来源”是 `run_events + artifacts + memory`。在 DSL/事件/Schema 快速收敛期过早拆分，会显著增加一致性、回放、幂等、运维与版本兼容成本，拖慢交付闭环成熟。
- **演进原则**：
  - 先“**边界清晰的进程外插件**”，再“**真正的服务拆分**”，最后才考虑消息总线/服务网格。
  - 拆分优先考虑：**安全隔离收益**、**吞吐扩缩收益**、**故障域隔离收益**。

## 12. 架构演进路线（从单体到分布式）

### 12.1 Stage 0：单体（M0/M1 默认）

- 形态：单个 `ash-worker`（Gin+GORM）负责编排、存储、SSE、规则/门禁、记忆与基础检索。
- 扩展：内置插件（Prometheus/console/sqlite-only）+ 进程外插件接口预留（Proto/gRPC 优先，Buf 管理 breaking）。
- 适用：单团队/内网、并发 run 较低、以快速迭代与语义收敛为主。

### 12.2 Stage 1：单体 + 进程外插件（推荐的“准分布式”）

> 不拆核心状态机，只把高成本/高风险能力外置，仍保持回放与一致性简单。

- **优先外置的插件（按收益排序）**
  - **Executor Service（工具执行/沙箱）**：把危险工具与执行环境隔离（安全收益最高）
  - **RAG Indexer/Retriever（索引与检索）**：资源型能力弹性扩缩，避免拖慢编排主循环
  - **Observability Exporter**：对外发指标/trace/log，避免主进程受外部链路影响
- 技术接口：**Proto/gRPC（Buf breaking check）** 为主，HTTP/MCP 为辅（按集成方生态选择）。

### 12.3 Stage 2：服务化拆分（P2：组织级/多租户）

- 前提：DSL/事件/Schema 基本稳定；回放一致性策略可验证；权限与审计要求提升。
- 典型拆分：
  - **Control Plane（编排/规则/门禁）**：Run 生命周期、策略、审计与回放控制
  - **Data Plane（执行/索引/检索）**：工具执行、索引构建、向量检索
  - **Memory Service**（可选独立）：组织级记忆治理、多租户与权限隔离
- 数据层演进：
  - 元数据：SQLite → **Postgres**
  - Artifacts：本地目录 → **对象存储（S3/MinIO）**
  - 向量库：本地/可选 → **Qdrant（优先）/Milvus（规模更大）**

### 12.4 Stage 3：事件驱动分布式（P3+，规模化与流水线）

- 触发：需要大量异步流水线（批量索引、评测、改进提案实验）与跨服务强解耦。
- 技术选型：
  - 消息总线：NATS（轻量）或 Kafka（规模化）
  - 一致性：Outbox/Inbox + 幂等键 + 重放策略（事件为真相）
- 说明：该阶段对团队工程能力要求高，不建议早期引入。

## 13. 架构升级的重要指标节点（触发阈值与门禁）

> 用指标驱动架构升级决策，避免“为拆而拆”。建议将这些指标纳入可观测性（Prometheus/OTel）并作为升级评审材料。

### 13.0 指标口径与采集来源（默认）

- **窗口**：
  - **告警/触发观察**：滚动 1h / 24h
  - **升级评审**：至少连续 7 天趋势 + 峰值日（高峰）数据
- **来源**：
  - Prometheus 指标（见 [`../appendices/D-Observability-指标与告警.md`](../appendices/D-Observability-指标与告警.md)）
  - run_events 离线重算报表（避免采样/外发影响口径）
- **Owner（默认）**：
  - 性能/容量：SRE/平台
  - 安全/合规：Security
  - 记忆治理：Librarian/平台

### 13.1 Stage 0 → Stage 1（外置插件）的触发指标（量化）

> 目标：不拆核心编排，只把“执行/检索/导出”外置以获得隔离与弹性。


| 触发类别          | 指标                         | 默认阈值（建议）                  | 统计窗口 | 数据来源                                | Owner       | 触发后的优先外置模块                       | 升级评审证据                |
| ------------- | -------------------------- | ------------------------- | ---- | ----------------------------------- | ----------- | -------------------------------- | --------------------- |
| **安全隔离**      | 高危工具需求出现频次                 | ≥ 1 次/周 且需要自动化执行          | 7d   | run_events（tool.called risk=danger） | Security    | Executor（沙箱）                     | 失败/阻断 runId 样本 + 审计记录 |
| **门禁阻塞**      | `policy.denied` 阻塞率        | ≥ 5% 的 runs 被阻断           | 7d   | Prometheus + 离线重算                   | Security/PM | Executor（human+沙箱）               | denied 原因 TopN + 影响场景 |
| **吞吐压力**      | `ash_run_inflight` 持续高位    | P95 > 20 且持续 > 1h         | 24h  | Prometheus                          | SRE         | RAG/Executor                     | inflight 与队列深度曲线      |
| **端到端延迟**     | Run P95 时延回归               | 相对基线 +30% 且持续 7d          | 7d   | Prometheus                          | SRE         | RAG Indexer/Retriever            | P95 对比（升级前后）          |
| **工具耗时**      | `ash_tool_duration_ms` P95 | 任一关键 tool P95 > 30s 持续 7d | 7d   | Prometheus                          | SRE         | Executor                         | tool 分布与慢点定位          |
| **检索耗时**      | `ash_rag_latency_ms` P95   | P95 > 2s 持续 7d            | 7d   | Prometheus                          | SRE         | RAG Retriever                    | topK/缓存命中率/慢查询样本      |
| **外部依赖影响主流程** | run 因导出/索引失败而失败            | ≥ 1% 的 runs               | 7d   | run.failed 归因                       | SRE         | Observability Exporter / Indexer | 失败归因报告（按 error.code）  |


### 13.1 Stage 0 → Stage 1（外置插件）的触发指标

- **安全隔离需求（任一满足即可）**
  - 需要执行高危命令/部署动作且要求与主进程隔离（合规/审计）
  - `policy.denied` 高频阻塞交付，需要引入“human step + 沙箱执行面”
- **性能/吞吐（任一满足）**
  - `ash_run_inflight` 长期高位（例如持续 > 20）且 P95 run 时延上升
  - `ash_tool_duration_ms` 或 `ash_rag_latency_ms` 的 P95 连续 1 周超过基线阈值
- **稳定性（任一满足）**
  - 外部依赖（检索/观测导出/索引）故障影响主流程（应外置隔离）

### 13.2 Stage 1 → Stage 2（服务化拆分）的触发指标

- **多租户/组织级治理**
  - 需要按 team/repo 隔离数据与权限（RBAC/ABAC），并要求独立审计与保留策略
- **数据规模**
  - `run_events`/audit/记忆库增长导致 SQLite 性能或运维不可接受（备份/归档/并发写冲突）
- **可用性/SLO**
  - run 成功率、恢复成功率、SSE 续传成功率无法稳定达标，且瓶颈明确在存储/执行/索引

### 13.3 Stage 2 → Stage 3（事件驱动）的触发指标

- **异步流水线需求**
  - 批量索引/评测/改进实验任务量大，需要与在线交付隔离并弹性扩缩
  - 对照实验/回放任务显著增长，必须队列化执行

### 13.4 升级门禁（架构变更必须满足）

- **回放一致性门禁**：TR0-03（以及 TR1/TR3 对应项）必须保持全绿
- **安全门禁**：TR2 用例集（注入/secret/权限矩阵）不得回归
- **数据迁移门禁**：至少 1 次 v1→v2 迁移演练可回滚（见 [`../appendices/I-GORM-模型映射与迁移策略.md`](../appendices/I-GORM-模型映射与迁移策略.md)）
- **成本门禁**：token/任务与 P95 延迟不应显著回归（定义阈值并记录基线）

### 13.5 Stage 1 → Stage 2（服务化拆分）的触发指标（量化）

> 目标：引入 Control Plane/Data Plane 分离与组织级能力，通常伴随 Postgres/对象存储/向量库服务化。


| 触发类别              | 指标                | 默认阈值（建议）                           | 统计窗口 | 数据来源                            | Owner     | 推荐拆分                           | 升级评审证据           |
| ----------------- | ----------------- | ---------------------------------- | ---- | ------------------------------- | --------- | ------------------------------ | ---------------- |
| **多租户治理**         | 需要隔离的 team/repo 数 | ≥ 5 个团队或 ≥ 20 个 repo               | 30d  | 需求/组织输入                         | 平台负责人     | Control Plane / Memory Service | 权限矩阵草案 + 审计与保留策略 |
| **SQLite 运维不可接受** | DB 锁冲突/写入失败       | 出现不可恢复写失败或频繁锁等待                    | 7d   | DB 日志 + run.failed              | SRE       | 元数据→Postgres                   | 故障复盘 + 迁移演练计划    |
| **事件增长**          | run_events 增长     | > 5M events/月 或单机存储压力显著            | 30d  | run_events 统计                   | SRE       | 事件归档/分区                        | 归档与分区方案          |
| **Artifacts 体量**  | artifacts 占用      | > 50GB/月 或清理影响生产                   | 30d  | 文件统计                            | 平台        | 对象存储                           | 成本估算 + 生命周期策略    |
| **记忆治理负担**        | 未评审 backlog       | L1 backlog > 500 持续 7d             | 7d   | `ash_memory_unreviewed_backlog` | Librarian | Memory Service 独立              | 队列与 SLA 报表       |
| **可用性达标困难**       | run 成功率           | Feature < 95% 或 Hotfix < 99% 持续 7d | 7d   | doctor 报告 + 指标                  | SRE       | 控制/执行分离                        | 失败归因 TopN        |


### 13.6 Stage 2 → Stage 3（事件驱动）的触发指标（量化）

> 目标：大规模异步流水线（批量索引/评测/实验）与在线交付隔离。


| 触发类别       | 指标              | 默认阈值（建议）     | 统计窗口 | 数据来源                  | Owner | 推荐引入          | 升级评审证据         |
| ---------- | --------------- | ------------ | ---- | --------------------- | ----- | ------------- | -------------- |
| **批处理规模**  | 索引/评测任务量        | ≥ 1000 job/天 | 7d   | 作业统计                  | 平台    | NATS/Kafka    | Job 类型与 SLA 需求 |
| **回放对照需求** | replay runs 占比  | > 20% 的总 run | 7d   | run_events（replay 标记） | 平台/研究 | 队列化执行         | 实验对照报告与资源评估    |
| **在线被拖慢**  | 在线 run 延迟受批处理影响 | 相关性显著（同峰期抖动） | 7d   | 指标关联分析                | SRE   | 异步总线 + outbox | 峰值日性能剖析        |


### 13.7 升级评审 Checklist（必须提交的材料）

- **背景与动机**
  - 触发指标命中情况（阈值、窗口、趋势图）
  - 当前架构瓶颈归因（存储/执行/检索/导出）
- **方案与边界**
  - 拆分边界（Control/Data/Memory/Indexer/Executor）
  - API 契约（OpenAPI + Proto）版本策略与兼容方案（buf breaking 报告）
- **数据与回放**
  - 事件与 artifacts 的迁移/归档策略
  - 回放一致性验证计划（TR0-03 与回归集）
- **安全与合规**
  - 权限矩阵变更、危险操作批准流
  - secret 脱敏与审计方案
- **运维与成本**
  - SLO/SLA 目标、扩缩容策略
  - 成本估算（存储/向量库/消息总线/计算）
- **验收门禁**
  - TR0 全绿、TR2 不回归、迁移可回滚（至少 1 次演练）