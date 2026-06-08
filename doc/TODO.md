# ASH 待办 / 技术债

> 记录尚未完成或需人工环境验证的项。完成请移入 CHANGELOG 并删除对应条目。

## 测试验收

> **状态**：Sprint I/H 完成后恢复执行；以下项以本地 `go test` 为准（T-05/T-06 仍需 Postgres）。
> **关联**：derive parity、Doctor TR3-05、`postgres-sql-schema-e2e` CI job

| # | 项 | 命令 / 入口 | 验收 |
|---|-----|-------------|------|
| T-01 | derive 回放 parity 单测 | `go test ./internal/observability/derive/... -run Parity -count=1` | ✅ 2026-06-03 本地 pass |
| T-02 | derive 记忆 backlog 单测 | `go test ./internal/observability/derive/... -run Replay_memory -count=1` | ✅ 2026-06-03 本地 pass |
| T-03 | Doctor TR3-05/06/07 | `go test ./internal/doctor/... -run TestTR3Suite -count=1` | ✅ TR3 **7/7**（TR3-06 sqlite skip） |
| T-04 | Doctor ALL 计数 | `go test ./internal/doctor/... -run TestALLSuite -count=1` | ✅ **36/36** pass |
| T-05 | SQL schema 本地 E2E | `make postgres-sql-schema-e2e` | ✅ 2026-06-08：`ASH_SCHEMA_MODE=sql`、sql rev **17**、M3 **8/8**（M3-04 skip） |
| T-06 | CI job | GitHub Actions `ci.yml` + nightly `postgres-e2e.yml` | PR/main `postgres-sql-schema-e2e`；nightly 全量 migrate e2e |
| T-13 | Postgres RAG FTS 集成 | `go test -tags=integration ./internal/rag/ -run TestPostgresRAGFTSQuery`（`ASH_MIGRATE_E2E=1`） | 已并入 `make postgres-sql-schema-e2e` |
| T-07 | 记忆 emit layer | `go test ./internal/memory/... -count=1 -short` | ✅ 断言 pass（Windows 偶发 TempDir 清理警告） |
| T-08 | Scale 双写冲突告警 | `go test ./internal/api/... -run ScaleReadinessSchemaSql -count=1` | ✅ 2026-06-03 本地 pass |
| T-09 | 记忆 P1 derive 单测 | `go test ./internal/observability/derive/... -run Replay_memoryHit -count=1` | ✅ 2026-06-03 本地 pass |
| T-10 | 记忆 schema 迁移 | `go test ./internal/memory/... -run RunMigrations -count=1` | ✅ 断言 pass（Windows TempDir 清理警告） |
| T-11 | OTel 骨架单测 | `go test ./internal/observability/otel/... -count=1` | ✅ 2026-06-03 本地 pass |
| T-12 | RAG 降级 derive | `go test ./internal/rag/... ./internal/observability/derive/... -run 'Fallback|ragRetrieved' -count=1` | ✅ Sprint I：`retrievalMode=chunk` replay |

```bash
# 一键回归（网络正常时）
go test ./internal/observability/derive/... ./internal/doctor/... ./internal/memory/... ./internal/api/... -count=1 -short
make postgres-sql-schema-e2e
```

---

## 人工验证（暂缓）

> **状态**：依赖云 RDS / 真实 GitHub token / live ExecGo，当前迭代**暂不执行**，保留清单待发布窗口统一验收。
> **清单**：[`doc/checklists/postgres-rds-e2e.md`](checklists/postgres-rds-e2e.md) · 一键脚本 `make postgres-rds-e2e`

| # | 项 | 命令 / 入口 | 验收 |
|---|-----|-------------|------|
| H-01 | 云 RDS 全链路 E2E | `make postgres-rds-e2e` | `migrate schema` v17 + Doctor M3 8/8 + ALL 36/36 |
| H-02 | 云 RDS RLS + ash_app | 清单 §4–§5 | M3-06/07 pass；`TestPostgresRLSE2EAfterMigrate` |
| H-03 | 生产 Worker 配置 | `ASH_DATABASE_APP_URL` + `ASH_POSTGRES_RLS_FORCE=1` | `/readyz` dialect=postgres |
| H-04 | GitHub CI runs 同步 | `GET /api/v1/ci/runs?sync=true` | 真实 token 拉取 Actions 摘要 |
| H-05 | GitHub CI jobs / 日志诊断 | `GET /api/v1/ci/jobs?runId=...&sync=true` + diagnose | job log 落库与 rootCause |
| H-06 | ExecGo live smoke | `ASH_EXECGO_E2E=1` + Doctor M3-05 | live 执行链路通过 |
| H-07 | 密钥轮换策略 | repo connection `secretId` | 接入团队 Secrets 轮换 SOP |
| H-08 | 发布窗口 audit gate | 真实发布 checklist | Postgres e2e + ALL/M3 + ExecGo 证据归档 |
| H-09 | 业务抽样 §7 | 清单 §7.1–7.7 | Run/SSE/Memory/KPI/CI/合规/Scale |

本地自动化（无需云环境，CI 可跑）：

```bash
make postgres-e2e
make postgres-rls-e2e
go test ./internal/metrics/... ./internal/doctor/... -count=1 -short
bash scripts/verify-local.sh
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

## 2. CI / ExecGo / KPI（MVP 已完成，待 H-04–H-06）

**状态**：API 与控制台 MVP 已交付；外部依赖验证见 **人工验证（暂缓）**
**优先级**：P0 自动化回归 / P1 人工闭环

```bash
go test ./internal/ci ./internal/metrics ./internal/api -run 'TestDiagnose|TestOverview|TestCreateRepo|TestRepoConnection' -count=1
make execgo-health
go run ./cmd/cli doctor --suite ALL --format md --agent static
make web-build
```

**已实现（自动化可验）**：

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

- 新表 DDL 修订需同步追加 RLS policy（`000013` DO 块）与 `ash_app` GRANT（`memory_migrations` 为全局审计表、无 `space_id`， intentionally 不纳入租户 RLS）

---

## 已完成（近期）

- Sprint Q：`/readyz` 运维快照（schema/otel/alerts/replay）；生产配置模板可观测性段；**CI PR** 接入 `postgres-sql-schema-e2e`
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
