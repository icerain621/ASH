# ASH 进度与里程碑（Plan）v0.1

> 文档状态：v0.1（可执行排期版）。以 M0/M1 为主，M2/M3 为展望。

## 1. 团队与产能假设
- **默认团队**：4 人（2 后端/平台 + 1 前端 + 1 全栈/测试）
- **有效产能**：约 100h/周（团队合计）
- **迭代节奏**：2 周/Sprint（≈200h/Sprint）

**TODO（负责人：项目经理）**：补全 RACI（每 Epic/Story 的 owner、reviewer、approver）。  
**验收方式**：Sprint 0 评审通过 RACI 表。

## 2. 里程碑定义
- **M0（P0）**：Feature 交付闭环 + 回放 + 记忆评审 + SSE 可观测 + Prometheus + `ash doctor(TR0)`
- **M1（P1）**：三场景模板齐全 + 引用门禁 + 记忆治理(去重/冲突) + OTel/质量插件 + 自我迭代 1.0 + `doctor(TR1/TR2 子集)`
- **M2（P2）**：组织级权限/合规/多租户 + 强隔离 + 存储演进
- **M3（P3）**：网关/技能市场/知识图谱增强

## 3. 关键路径（M0）
事件与回放 → DSL → workflow+checkpoint → toolbus → memory → obs+doctor

## 4. Sprint 计划（M0）
> 详细 Backlog 与验收矩阵见 `docs/appendices/`。

### Sprint 1（事件流 + DSL 可加载 + 最小 gate）
- 交付：SSE 续传；DSL 校验加载；git.status gate

### Sprint 2（能改代码 + 能跑测试 + checkpoint 初版）
- 交付：apply_patch/test_report；Feature 模板基本闭环；checkpoint per_step

### Sprint 3（记忆评审 + obs 插件 + doctor TR0）
- 交付：candidate→review→merge；Prometheus；doctor(TR0) 全绿；Web 最小闭环

## 5. 验收与门禁
- **TR0（M0）**：闭环/事件/回放
- **TR1/TR2（M1）**：provider 替换、schema 校验、checkpoint 恢复、注入红队、secret 扫描、MCP 隔离

## 6. 风险登记（摘要）
- DSL 语义膨胀、checkpoint 语义、apply_patch 可靠性、跨平台工具链、RAG 引用质量、记忆评审吞吐、门禁误杀、观测泄密风险

**TODO（负责人：项目经理）**：补全 Risk Register（触发阈值/回滚动作/owner）。  
**验收方式**：每 Sprint Review 更新一次并追踪趋势。

## 7. 实现进度（代码仓）
> 代码目录：`ash/`（Go Worker，Gin + GORM + SQLite）

| 模块 | 状态 | 说明 |
|---|---|---|
| Worker 骨架（Gin） | ✅ 已启动 | `ash/cmd/worker`，health/ready/metrics |
| GORM + SQLite 模型 | ✅ 已启动 | runs、run_events、memory/audit 表 AutoMigrate |
| Runs API | ✅ 已启动 | POST/GET runs，创建时写入 run.started/finished 事件 |
| Run 控制 (resume/replay) | ✅ M0 | `POST /runs/:id/resume`、`POST /runs/:id/replay`；`run.json` 持久化 |
| SSE 事件流 | ✅ 已启动 | `/api/v1/runs/:id/stream`，支持 Last-Event-ID |
| Rules DSL 引擎 | ✅ 已启动 | 解析/校验/Loader/Engine；`feature_delivery` / `hotfix` / `security_patch` |
| Scenarios API | ✅ 已启动 | list/get/validate |
| Runs 绑定 DSL | ✅ 已启动 | 按 steps 发 step/tool/checkpoint 事件 |
| ToolBus / Policy | ✅ M0 | git.status/diff/checkout、apply_patch、test.run |
| Artifacts 四件套 | ✅ M0 | manifest + diff/test_report/release/rollback |
| ash doctor (TR0) | ✅ M0 | `go run ./cmd/cli doctor --suite TR0` 或 POST `/api/v1/doctor/run` |
| Memory Review API | ✅ M0 | candidates CRUD/review、query、hit-used、audit |
| Web UI | ✅ M0 React | Vite+React+TS；`make web-build` → `frontend/dist`；Worker `/ui/` |
| 目录结构 (monorepo) | ✅ 脚本就绪 | `make reorganize` 或 `scripts/reorganize.ps1` → `backend/` + `doc/design/` |
| Swagger (swag) | ✅ M0 已实现端点 | `/docs`、`/openapi.json`；`make swagger` 再生 |
| Proto/gRPC 插件 ABI | ✅ M0 基础 | HTTP 注册 + 可选 gRPC（`ASH_PLUGIN_GRPC_ADDR`，dev 默认 `127.0.0.1:19091`） |
| 控制台场景选择 | ✅ M0 | Runs 页从 `/api/v1/scenarios` 选择场景创建运行 |
| Doctor TR0-08 | ✅ M0 | 三场景模板加载校验（`TR0-08`） |
| 引用门禁 (requireCitations) | ✅ M1 | 缺失引用阻断或降级人工确认；质量指标 `citation_*` |
| 记忆治理 (去重/冲突) | ✅ M1 | 评审自动建边；创建候选返回 `governance` 提示 |
| 自我迭代 1.0 | ✅ M1 基础 | `/api/v1/improve/proposals` 提案→回放对照→灰度→晋升/回滚 |
| Doctor TR1 | ✅ M1 | TR1-01..05（路由/瀑布/记忆冲突/MCP/DSL 校验） |
| 控制台 M1 | ✅ 进行中 | Memory 治理边 + 检索；Automation 自我迭代面板 |
| TR2 合规控制台 | ✅ M1 | `/ui/compliance`：TR2 检查卡、身份/作用域/存储/插件、诊断报告表 |
| 资源作用域 API | ✅ TR2 | `GET /api/v1/spaces/:spaceId/resource-scopes` |
| Secret 泄漏扫描 | ✅ TR2 | `GET /api/v1/compliance/secret-scan`；审计列表 RedactJSON |
| TR2-05 Doctor | ✅ TR2 | 泄漏探测 + 载荷脱敏校验 |
| 合规审计导出 | ✅ TR2 | `POST /api/v1/compliance/export`（日志 + Doctor 报告） |
| Doctor ALL 套件 | ✅ 修复 | `suite=ALL` 运行 TR0+TR1+TR2+TR3 全量用例 |
| Doctor TR3 | ✅ GA 基础 | TR3-01..04（记忆迁移/RAG 降级/SLO/审计溯源） |
| TR3 规模化控制台 | ✅ GA 基础 | `/ui/scale` + `GET /api/v1/scale/readiness` |
| Run 溯源 API | ✅ TR3 | `GET /api/v1/runs/:runId/provenance` |
| 合规 Redact 快捷操作 | ✅ TR2 | 合规页一键 `PUT /audit/policy` 开启脱敏 |
| 导出含 Secret 扫描 | ✅ TR2 | 合规导出 JSON 内嵌 `secretScan` 摘要 |
| Runs 溯源面板 | ✅ TR3 | 运行详情展示 `/runs/:id/provenance` 链路 |
| OpenAPI 合规/规模化/迭代 | ✅ | `bash scripts/regenerate-swagger.sh` 含 improve/compliance/scale/provenance |
| Doctor ALL 回归测试 | ✅ | `TestALLSuite`（30 项，非 `-short`，含 M3-05 ExecGo live smoke skipped/live 门禁） |
| Runs 控制台 actorRole | ✅ | 创建运行可选角色；API 返回 `executionError` |
| 跨空间访问 403 | ✅ | `requireRequestSpace` 统一 runs/space-param/secrets/RAG/MCP/审批等 |
| M2 权限矩阵 | ✅ 基础 | `internal/authz` + API/UI + 场景工具 enforcement + Doctor M2-01 |
| M2 场景策略 API | ✅ | `PUT /spaces/:id/resource-scopes/:scopeId` + 审计；M2-02/03；ALL 27 项 |
| M2 运行期 enforcement | ✅ | `m2_policy_enforce` 场景 + `POLICY_DENIED` + `policy.denied` 事件 |
| M3 多租户 / Postgres | ✅ 展望 | Doctor M3-01..04；`docker-compose.postgres.yml`；`make postgres-e2e`；`ash migrate` CLI |
| GitHub Actions CI 门禁 | ✅ PRD 前四项 | PR/main 执行 Go + Doctor static + Web build；manual/nightly 执行 Postgres e2e |
| ExecGo/Codex 执行面稳定化 | ✅ PRD 前四项 | `make execgo-health` 分类失败；`ASH_EXECGO_E2E=1` 触发 Doctor M3-05 live smoke |
| Repo/CI 连接与诊断 | ✅ PRD 前四项 | GitHub Actions provider；`/api/v1/repo/connections`、`/api/v1/ci/runs`、`/api/v1/ci/failures/diagnose` |
| KPI 指标看板 | ✅ PRD 前四项 | `/api/v1/metrics/overview` + `/ui/metrics`；缺失 SSE 数据源返回 unavailable |
| SQLite 纯 Go 驱动 | ✅ | `glebarez/sqlite`，Windows 无需 CGO |
| 本地验证脚本 | ✅ | `bash scripts/verify-local.sh`（Git Bash / Linux） |

本地验证：
```bash
bash scripts/verify-local.sh
make tidy && make web-build && make run
make test
make swagger
make doctor
go run ./cmd/cli doctor --suite TR1 --format md
go run ./cmd/cli doctor --suite TR3 --format md
go run ./cmd/cli doctor --suite M3 --format md --agent static
go test ./internal/ci ./internal/metrics ./internal/api -run 'TestDiagnose|TestOverview|TestCreateRepo|TestRepoConnection' -count=1
# Run control
curl -X POST http://localhost:8080/api/v1/runs/{runId}/resume
curl -X POST http://localhost:8080/api/v1/runs/{runId}/replay -H "Content-Type: application/json" -d "{\"mode\":\"exact\"}"
```
