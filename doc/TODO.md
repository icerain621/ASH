# ASH 待办 / 技术债

> 记录尚未完成或需人工环境验证的项。完成请移入 CHANGELOG 并删除对应条目。

## 人工验证（暂缓）

> **状态**：依赖云 RDS / 真实 GitHub token / live ExecGo，当前迭代**暂不执行**，保留清单待发布窗口统一验收。
> **清单**：[`doc/checklists/postgres-rds-e2e.md`](checklists/postgres-rds-e2e.md) · 一键脚本 `make postgres-rds-e2e`

| # | 项 | 命令 / 入口 | 验收 |
|---|-----|-------------|------|
| H-01 | 云 RDS 全链路 E2E | `make postgres-rds-e2e` | `migrate verify` + Doctor M3 7/7 + ALL 32/32 |
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

（暂无 — 见下方已完成与人工验证暂缓）

---

## 已完成（近期）

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
