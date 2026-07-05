# ASH 待办 / 技术债

> 记录尚未完成或需人工环境验证的项。完成请移入 CHANGELOG 并删除对应条目。

## 测试验收

> **状态**：Sprint I/H 完成后恢复执行；以下项以本地 `go test` 为准（T-05/T-06 仍需 Postgres）。
> **关联**：derive parity、Doctor TR3-05、`postgres-sql-schema-e2e` CI job

| # | 项 | 命令 / 入口 | 验收 |
|---|-----|-------------|------|
| T-01 | derive 回放 parity 单测 | `go test ./internal/observability/derive/... -run Parity -count=1` | ✅ 2026-06-03 本地 pass |
| T-02 | derive 记忆 backlog 单测 | `go test ./internal/observability/derive/... -run Replay_memory -count=1` | ✅ 2026-06-03 本地 pass |
| T-03 | Doctor TR3-05..10 | `go test ./internal/doctor/... -run TestTR3Suite -count=1` | ✅ TR3 **10/10**（TR3-06/08 默认 skip） |
| T-04 | Doctor ALL 计数 | `go test ./internal/doctor/... -run TestALLSuite -count=1` | ✅ **43/43** pass |
| T-05 | SQL schema 本地 E2E | `make postgres-sql-schema-e2e` | ✅ sql rev **20**；TR3-06/10 live 断言（M3-04 skip by design） |
| T-06 | CI job | GitHub Actions `ci.yml`（`regression-short` + 三门 Postgres + **`postgres-e2e` PR** + **`web-gate` frontend**）+ `postgres-e2e.yml` nightly | PR 四门 Postgres + 全量 migrate e2e |
| T-13 | Postgres RAG FTS 集成 | `go test -tags=integration ./internal/rag/ -run TestPostgresRAGFTSQuery`（`ASH_MIGRATE_E2E=1`） | 已并入 `make postgres-sql-schema-e2e` |
| T-14 | Memory 子表 RLS 集成 | `go test -tags=integration ./internal/store/ -run TestPostgresRLSSpaceIsolationOnMemoryChildren` | 已并入 `postgres-rls-e2e` / `postgres-sql-schema-e2e` |
| T-15 | Org 身份 RLS 集成 | `go test -tags=integration ./internal/store/ -run TestPostgresRLSSpaceIsolationOnOrgIdentity` | 已并入 `postgres-rls-e2e`；`verify-local` Docker 可用时自动跑 |
| T-07 | 记忆 emit layer | `go test ./internal/memory/... -count=1 -short` | ✅ 断言 pass（OpenTest 关闭 DB） |
| T-08 | Scale 双写冲突告警 | `go test ./internal/api/... -run ScaleReadinessSchemaSql -count=1` | ✅ 2026-06-03 本地 pass |
| T-09 | 记忆 P1 derive 单测 | `go test ./internal/observability/derive/... -run Replay_memoryHit -count=1` | ✅ 2026-06-03 本地 pass |
| T-10 | 记忆 schema 迁移 | `go test ./internal/memory/... -run RunMigrations -count=1` | ✅ OpenTest 关闭 DB |
| T-11 | OTel 骨架单测 | `go test ./internal/observability/otel/... -count=1` | ✅ 2026-06-03 本地 pass |
| T-12 | RAG 降级 derive | `go test ./internal/rag/... ./internal/observability/derive/... -run 'Fallback|ragRetrieved' -count=1` | ✅ Sprint I：`retrievalMode=chunk` replay |

```bash
# 一键回归（网络正常时）
make regression-short
go test ./internal/observability/derive/... ./internal/doctor/... ./internal/memory/... ./internal/api/... -count=1 -short
make postgres-sql-schema-e2e
```

---

## 人工验证（暂缓）

> **状态**：依赖云 RDS / 真实 GitHub token / live ExecGo，当前迭代**暂不执行**，保留清单待发布窗口统一验收。
> **清单**：[`doc/checklists/postgres-rds-e2e.md`](checklists/postgres-rds-e2e.md) · 一键脚本 `make postgres-rds-e2e`

| # | 项 | 命令 / 入口 | 验收 |
|---|-----|-------------|------|
| H-01 | 云 RDS 全链路 E2E | `make cloud-acceptance`（需 `ASH_DATABASE_URL`） | `migrate schema` v20 + Doctor M3 11/11 + ALL 43/43 |
| H-02 | 云 RDS RLS + ash_app | 清单 §4–§5；本地 `make postgres-app-gate` | M3-06/07 pass；`TestPostgresRLSE2EAfterMigrate` |
| H-03 | 生产 Worker 配置 | `ASH_DATABASE_APP_URL` + `ASH_POSTGRES_RLS_FORCE=1` | `/readyz` dialect=postgres；签字见 `h01-h03-cloud-signoff.md` |
| H-04 | GitHub CI runs 同步 | `GET /api/v1/ci/runs?sync=true` | 真实 token 拉取 Actions 摘要；本地/CI 可用 **`ASH_CI_FIXTURE=1`**（`TestCISyncRunsWithFixture` / `TestReleaseSamplingCIFixtureH04H05`） |
| H-05 | GitHub CI jobs / 日志诊断 | `GET /api/v1/ci/jobs?runId=...&sync=true` + `POST /ci/failures/diagnose`（仅 `jobId`） | job log 经 fixture 拉取、`logDigest` 落库、`rootCause` 分类；`TestFixtureProviderSyncRunsAndJobs` |
| H-06 | ExecGo live smoke | `ASH_EXECGO_E2E=1 make execgo-live-smoke` 或 Doctor M3-05 | `execgo-health` + live 执行链路；本地静态 `TestM3ExecGoLiveSmoke` |
| H-07 | 密钥轮换策略 | `make secret-rotate-smoke` 或 `TestSecretRotateRepoConnectionH07` | secret 轮换后 repo connection + CI sync 仍可用 |
| H-08 | 发布窗口 audit gate | [`release-window-audit.md`](checklists/release-window-audit.md) + `make release-window-audit` | Postgres e2e + ALL/M3 + §7 证据归档 |
| H-09 | 业务抽样 §7 | `make release-sampling-static` 或 `make release-sampling-smoke` | Run/SSE/Memory/KPI/CI/合规/Scale（§7.2 SSE 含 `TestReleaseSamplingSSE`） |

本地自动化（无需云环境，CI 可跑）：

```bash
make cloud-acceptance      # H-01~H-03 云 RDS + 证据归档（需 ASH_DATABASE_URL）
make mvp-signoff           # MVP 发布签字门禁
make web-gate              # 前端 lint + vitest + build
make production-config-gate
make rollback-drill
make queue-gate
make t0-alert-gate
make data-backup            # SQLite 备份（migrate 前）
make postgres-e2e
make postgres-rls-e2e
make postgres-app-gate    # Docker + schema 后 H-02/H-03 本地
make smoke-static
bash scripts/verify-local.sh
# Live Worker（可选）
ASH_WORKER_URL=http://127.0.0.1:8080 ASH_CI_FIXTURE=1 make live-smoke
```

---

## 1. Postgres 迁移（代码已完成，待 H-01/H-02）

**状态**：本地 `make postgres-e2e` 已通过；生产切换见 **人工验证（暂缓）** H-01–H-03
**优先级**：P1（切换生产前必做）

```bash
make postgres-e2e
make postgres-roles
make test-integration   # 需 ASH_DATABASE_URL + ASH_MIGRATE_E2E=1
```

**自动化验收**（本地 / CI nightly）：

- `migrate verify` 全表行数一致
- Doctor **M3-04** 在 `ASH_MIGRATE_E2E=1` 时通过
- `TestPostgresReadyzProbe` → `dialect=postgres`
- `doctor --suite ALL` 无回退

---

## 2. CI / ExecGo / KPI（MVP 已完成，H-04–H-09 本地自动化已就绪）

**状态**：API 与控制台 MVP 已交付；外部依赖验证见 **人工验证（暂缓）**
**优先级**：P0 自动化回归 / P1 人工闭环

```bash
go test ./internal/ci ./internal/metrics ./internal/api -run 'TestDiagnose|TestOverview|TestCreateRepo|TestRepoConnection' -count=1
make execgo-health
go run ./cmd/cli doctor --suite ALL --format md --agent static
make web-build
```

**已实现（自动化可验）**：

- `ASH_CI_FIXTURE=1`：`TestCISyncRunsWithFixture` + `internal/ci` fixture provider（无需 GitHub token）
- `/readyz` `liveGateHints`：M3-04/05/06/07/08 与 CI fixture 状态
- GitHub Actions PR/main：Go + Doctor static + Web build
- Postgres e2e：nightly/manual workflow
- Repo `secretId` only；CI diagnose API；KPI overview；控制台 feedback/ci/observability/releases

---

## 3. Postgres RLS 生产部署（待 H-02/H-03）

**状态**：RLS 骨架与 API 接线已完成；生产角色与连接串见 **人工验证（暂缓）**
**优先级**：P2

```bash
make postgres-up
make postgres-roles
```

---

## 代码待办（可继续开发）

- 新表 DDL 修订需同步追加 RLS policy（`000013` 租户 / `000018` run / `000019` memory / `000020` org 扩展）；迁移目录表已全覆盖（`PostgresRLSDeferredTables()` 为空）；Doctor **M3-11** 校验

---

## 已完成（近期）

- Sprint BB：`make worker-local-gate` / `make pre-migrate-gate` / `make t1-metrics-gate`；`release-window-runbook.md`。
- Sprint BA：`make queue-gate` / `make t0-alert-gate` / `make data-backup`；`doc/mvp-release-scope.md`；CI production-config + queue gate。
- Sprint AZ：`make production-config-gate` + `make rollback-drill`；API 延迟/并发基线单测；`regression-short` / `mvp-signoff` 接入。
- Sprint AY：`make web-gate`（eslint + vitest + build）；CI frontend job 升级；`source-cloud-rds-env.sh`；MVP §3 前端项勾选。
- Sprint AX：`make cloud-acceptance` + `make mvp-signoff` 证据归档；`h01-h03-cloud-signoff.md`；MVP 清单勾选与 `doc/evidence/`。
- Sprint AW：`scripts/postgres-app-gate.sh` + `make postgres-app-gate`（H-02/H-03 本地）；`smoke-static.sh`；`verify-local` Go env 引导。
- Sprint AV：`postgres-rds-e2e` 接入 `live-smoke`；`smoke-index` 文档串联；MVP 清单与 CI `release-sampling-static` 显式步骤。
- Sprint AU：`scripts/live-smoke.sh` + `make live-smoke`（H-04/05/06/07/09 live 编排）；`doc/checklists/smoke-index.md`；`release-window-audit` live 段收敛。
- Sprint AT：`scripts/release-sampling-static.sh` + `make release-sampling-static`（H-09 §7 静态烟测）；`release-sampling-smoke.md`；`regression-short` 接入。
- Sprint AS：`scripts/secret-rotate-smoke.sh` + `make secret-rotate-smoke`（H-07）；`TestSecretRotateRepoConnectionH07`；`make release-sampling`；`regression-short` / CI / `release-window-audit` 接入。
- Sprint AQ：`scripts/release-window-audit.sh` + `make release-window-audit`（H-08 静态门禁聚合）。
- Sprint AR：`scripts/execgo-live-smoke.sh` + `make execgo-live-smoke`（H-06）；`TestM3ExecGoLiveSmoke`；`verify-local` 静态门禁。
- Sprint AP：`scripts/ci-fixture-smoke.sh`  live Worker H-04/H-05；`release-sampling.sh` §7.3b ttl-queue；CI 页 fixture 提示 +「诊断选中 job」。
- Sprint AO：CI fixture H-04/H-05 全链路扩展（双 job：test_failure + docker）；`jobId` 诊断拉日志 + `logDigest` 落库 + adopt；`TestReleaseSamplingCIFixtureH04H05`。
- Sprint AN：OpenAPI 契约补全 TTL 端点/schema；`memoryTTLSweepInterval` 与 Scale TTL 字段对齐；`TestMemoryTTLQueueAndSweepAPI`；derive `memory.ttl_expired` replay；H-09 抽样含 `ttl-queue`。
- Sprint AM：Worker 后台记忆 TTL sweep（`ASH_MEMORY_TTL_SWEEP_INTERVAL`）；`/readyz` 与 Scale 暴露 `memoryTTLSweepInterval`。
- Sprint AJ：记忆 catalog v1→v2；`release-window-audit.md`；`release-sampling.sh` + API 抽样测试
- Sprint AD：Scale `/readyz` 运维面板；Postgres readyz+RLS 集成测试；CI PR `postgres-rls-e2e`
- Sprint AC：Doctor **TR3-10** readyz HealthResponse 契约；M3-09 SQL 修订预期校验；RDS 脚本 rev 20；ALL **42/42**
- Sprint AB：`/readyz` RLS/SQL 漂移信号；Doctor **M3-09** RLS catalog 证据；CI `postgres-rls-e2e` job
- Sprint AA：Scale RLS 预期策略数 + catalog 摘要 + readiness 漂移告警；RLS 新表清单；`regression-short` store 冒烟
- Sprint Y：memory 子表 RLS 集成测试 + e2e 脚本接入
- Sprint W：SQL rev **18** `model_usage` run-scoped RLS；`PostgresRLSDeferredTables()` / `VerifyRLSMigrationSQL()`；Doctor **M3-11**；附录 G §9 OpenAPI
- OpenAPI：补全 legacy `/v1/*` 规划路径具体 schema；`TestApiV1SuccessResponsesAvoidGenericEnvelope` 防回归；76+96 schema 字段对齐
- Sprint U：Doctor **M3-10** RLS 全局表排除（`memory_migrations`/`schema_meta`）；`PostgresRLSGlobalTables()` 目录；`verify-local` 接入 `regression-short`
- Sprint T：Doctor **TR3-08** Prometheus replay 段（`ASH_METRICS_EVENT_REPLAY=1`）；`make regression-short` 快捷回归；`/readyz` 默认运维标志断言
- Sprint S：记忆测试 `store.OpenTest`；Doctor UI M3-03..M3-09 中文标签
- Sprint R：`internal/opsenv` 共享运维快照；Doctor **M3-09** readyz/Scale 契约；CI push main 跑 `postgres-e2e`
- Sprint P：Doctor **TR3-07** 插件导出健康；Scale `metricsEventReplayEnabled`；Observability OTel 导出器健康卡片；`verify-local` 补 pluginhealth/alerts
- Sprint O：OTel waterfall 导出自动写入 `pluginhealth`（`ash-otel-exporter`）；Scale 暴露 `otelEnabled` / `alertsEvalInterval`；`postgres-e2e` 追加 live Postgres **TR3**
- Sprint N：Worker 后台治理告警评估（`ASH_ALERTS_EVAL_INTERVAL`）；`postgres-sql-schema-e2e` 追加 live Postgres **TR3**；Scale TR3-06 卡片；swagger 再生
- Sprint M：`postgres-e2e-migrate` 复用 `source postgres-up.sh`；`postgres-sql-schema-e2e` 追加 `TestPostgresRAGFTSQuery`；Doctor **TR3-06**（Postgres `tsvector` FTS，sqlite skip）
- Sprint L：Observability 治理指标告警表；Scale 插件导出健康摘要
- Sprint K：Postgres RAG `tsvector` + GIN（SQL **17**）；`postgres-tsvector` FTS 引擎；集成测试 `TestPostgresRAGFTSQuery`
- Sprint J：治理告警规则（记忆积压 / RAG FTS 降级率 / 插件导出失败）；Prometheus live 治理指标；`GET /api/v1/rag/profile`；Observability RAG 卡片
- Sprint H：`plugin.export_failed` / `plugin.export_reported` 审计事件；可选 `runId` 写 run 事件；derive `ash_plugin_export_failures_total`
- Sprint I：RAG `retrievalMode`（fts/chunk/empty）、`rag.retrieved` 事件、derive `ash_rag_queries_total` / `ash_rag_fts_fallback_total`、Scale RAG 检索模式行
- Sprint G：插件导出健康（`plugin_registry` 列 + SQL **16**、`pluginhealth.RecordExport`、`POST /plugins/{id}/export-report`、`GET /plugins/health`、Automation UI 健康列）
- Sprint F：OTel 骨架（`internal/observability/otel`、Worker `Init`、Run live span + waterfall OTLP 导出、`GET /observability/otel/status`、Observability UI OTel 卡片）
- Sprint E：`memory_migrations` 表 + `POST /api/v1/memory/migrate`；`memory.migrated` → `ash_memory_migration_runs_total`；Scale 记忆迁移按钮；SQL revision **15**
- Sprint D：记忆 P1 derive（`hit_used` / `deprecated` / `query`）；`memory.query` 事件（`runId` 可选）；TR0 payload schema 补全
- Sprint C：记忆治理 derive 指标 `ash_memory_unreviewed_backlog` / `ash_memory_missing_evidence_total`；`memory.reviewed` emit 补 `layer`；Scale `readinessWarnings`（`ASH_SCHEMA_MODE=sql` + 双写冲突）
- 测试验收 T-01..T-08 写入 **测试验收（暂缓）**，不阻塞后续开发
- Sprint A/B：CI `postgres-sql-schema-e2e`、derive `ValidateReplayParity`、Doctor **TR3-05**、生产配置模板与 RDS 清单对齐 revision 14
- RLS SQL 化：`000013` 租户策略（`ash_rls_*` 函数 + `ash_space_*` policies）、`000014` `ash_app`/`ash_rls_tester` 授权；运行时仅 backfill/FORCE
- `ASH_SCHEMA_MODE=sql` 试点：`make postgres-sql-schema-e2e`、Doctor **M3-08**、Scale readiness 暴露 schema 模式与 SQL 修订版本
- SQL 修订 `000009`–`000012` 完成迁移目录全表覆盖（CI/告警/发布/密钥/审批/插件/improve）；`expectedVersion=12`
- SQL 修订 `000005`–`000008`：run 执行（tool_calls/agent_tasks/artifact_index/checkpoints）、memory/RAG、租户身份、model/feedback
- 事件 → 指标表驱动派生：`internal/observability/derive`（附录 D §4 catalog + replay）；`ASH_METRICS_EVENT_REPLAY=1` 追加离线 replay 段
- SQL 修订 `000004`（run_steps / memory_records / audit_log）
- TR0 事件 payload JSON Schema 运行时校验（`ASH_VALIDATE_EVENT_PAYLOADS=1`）；SQL 修订 `000003`（runs/run_events）
- `golang-migrate` 版本化迁移骨架：Postgres embed SQL、`ash migrate schema`、`ASH_SCHEMA_MODE`、Doctor M3-03 校验
- Observability 配置 JSON Schema：`ash.obs/v0.1` + `config/ash-observability.yaml` + Worker 启动校验
- Rules DSL JSON Schema：`internal/rules/schemas/ash.rules.v0.1.schema.json` + `ValidateSchema`（20 非法样例回归）
- `runs.Service.eventsFor` 无限递归修复（审批/取消路径 stack overflow）；`openapi-check` 网络受限时自动重试（GOSUMDB=off + GOPROXY 镜像）
- API 错误码表：`doc/api/error-codes.md`、`internal/apicodes` 目录校验；CI 接入 `make openapi-check`
- OpenAPI 对齐：`doc/api/openapi-alignment.md`、`internal/openapicheck`、`make openapi-check`；手写契约已补全全部 `/api/v1` 实现端点
- Scale readiness：`lastMigrationSyncError` / `lastMigrationSyncErrorAtMs`；`migrate sync` 失败写入 `SyncState` 并成功后清空
- Scale readiness：`workerConnectionRole`、`runtimeDsnHint`、双写影子库脱敏 URL；M3 §3.4 PgBouncer 指引
- `metrics` 服务 `WithContext` + `metricsFor(c)`；OpenAPI 草稿补充 `/metrics/prometheus`
- TODO 重组：人工验证 H-01..H-09 单列暂缓
- KPI-07/KPI-10：`run_events` / `run_steps` 经 `runs.space_id` 租户过滤；`make postgres-rds-e2e`
- `/metrics` v1：全局 scrape + RLS bypass；`GET /api/v1/metrics/prometheus` 租户范围
- Postgres RLS + `ash_app` + Doctor M3-06/07 + e2e 脚本
- 云 RDS 清单 `doc/checklists/postgres-rds-e2e.md`
- PRD 前四项 + 后四项 MVP
