# 云 RDS Postgres E2E 验证清单

> 文档状态：可执行检查清单。在切换生产 `ASH_DATABASE_URL` / `ASH_DATABASE_APP_URL` 前，于目标云 RDS（AWS RDS、阿里云 RDS、自建云 Postgres 13+）按阶段执行。
>
> 本地 Docker 等价流程见 `make postgres-e2e`（`scripts/postgres-e2e-migrate.sh`）。云上无 `ash-postgres-dev` 容器，需按本文手工串联。

## 0. 前置条件（一次性）

| # | 项 | 验收 |
|---|-----|------|
| 0.1 | Postgres **13+**，库名如 `ash` | `psql` 可连 |
| 0.2 | **迁移账号** `ash`（或等价）：`CREATE` schema/table、`ALTER` RLS、建角色 | 能执行 DDL |
| 0.3 | **应用账号** `ash_app`：`NOBYPASSRLS`，仅 DML | 见 `scripts/postgres-init/01-ash-roles.sql` |
| 0.4 | SSL：`sslmode=require`（或 `verify-full` + CA） | URL 可 ping |
| 0.5 | 安全组 / 白名单：运维机 + Worker 出口 IP | 双向可达 |
| 0.6 | 源 SQLite 快照（`.ash/ash.db` 或备份） | 可 `migrate plan` |
| 0.7 | 密钥不入库：Secrets Manager / 环境注入 | 连接串不进 git |

**环境变量模板：**

```bash
# 迁移 / Doctor / 安装 RLS（owner）
export ASH_DATABASE_URL='postgres://ash:<MIGRATOR_PW>@<rds-host>:5432/ash?sslmode=require'

# Worker 运行时（生产）
export ASH_DATABASE_APP_URL='postgres://ash_app:<APP_PW>@<rds-host>:5432/ash?sslmode=require'

export ASH_DATA_DIR='.ash'
export ASH_SQLITE_PATH="$ASH_DATA_DIR/ash.db"
export ASH_MIGRATE_E2E=1
export ASH_POSTGRES_RLS=1
export ASH_POSTGRES_RLS_FORCE=1
export ASH_SCHEMA_MODE=sql
```

> **生产密码**：勿使用 dev 默认 `ash` / `ash_app`。在 RDS 上设置强密码后，同步更新 URL；角色与授权见 SQL 修订 `000014` 或 `01-ash-roles.sql`。  
> **生产配置模板**：[`postgres-production-config.md`](postgres-production-config.md)

---

## 1. 连通性与解析（约 5 分钟）

```bash
cd <ash-repo>
bash scripts/postgres-smoke.sh
```

| # | 检查 | 通过标准 |
|---|------|----------|
| 1.1 | URL 解析单测 | `TestParseDatabaseTargetPostgresURL` 通过 |
| 1.2 | Live 连接 | `doctor --suite M3` 中 **M3-02** pass（`dialect=postgres`） |
| 1.3 | 延迟 | `psql` ping 可接受（同区域建议 &lt;50ms）或记录基线 |

---

## 2. Schema 与角色（约 10 分钟）

```bash
export ASH_SCHEMA_MODE=sql

# 建表（golang-migrate 000001–000020，空库）
go run ./cmd/cli migrate schema up --postgres "$ASH_DATABASE_URL"
go run ./cmd/cli migrate schema version --postgres "$ASH_DATABASE_URL"
# 期望 version=20 expected=20 mode=sql

go run ./cmd/cli doctor --suite M3

# 应用角色（000014 已含 ash_app；或脚本兜底）
bash scripts/postgres-ensure-app-role.sh
```

| # | 检查 | 通过标准 |
|---|------|----------|
| 2.1 | 表目录 | **M3-03** pass，catalog ≥43 表；**M3-08** `sqlVersion=20` |
| 2.2 | RLS SQL | **M3-06** pass，`rlsPolicies` ≥41（修订 000013–000020） |
| 2.3 | `ash_app` 存在 | `\du ash_app` 或 M3-07 ping ok |
| 2.4 | 密码策略 | 生产强密码已轮换，`ASH_DATABASE_APP_URL` 已更新 |

---

## 3. 数据迁移 E2E（约 15–60 分钟，视数据量）

```bash
export ASH_MIGRATE_E2E=1
export ASH_DATA_DIR='.ash'
export ASH_SQLITE_PATH="$ASH_DATA_DIR/ash.db"

go run ./cmd/cli migrate plan \
  --data-dir "$ASH_DATA_DIR" \
  --sqlite "$ASH_SQLITE_PATH" \
  --postgres "$ASH_DATABASE_URL"

go run ./cmd/cli migrate copy \
  --data-dir "$ASH_DATA_DIR" \
  --sqlite "$ASH_SQLITE_PATH" \
  --postgres "$ASH_DATABASE_URL"

go run ./cmd/cli migrate verify \
  --data-dir "$ASH_DATA_DIR" \
  --sqlite "$ASH_SQLITE_PATH" \
  --postgres "$ASH_DATABASE_URL"
```

| # | 检查 | 通过标准 |
|---|------|----------|
| 3.1 | `migrate plan` | 各表行数对比无意外 diff（或已评审） |
| 3.2 | `migrate copy` | 退出码 0，无 FK / 唯一约束错误 |
| 3.3 | `migrate verify` | **全表行数一致**，退出码 0 |
| 3.4 | **M3-04** | `go run ./cmd/cli doctor --suite M3` → M3-04 pass（非 skipped） |
| 3.5 | 集成测试（可选） | `ASH_MIGRATE_E2E=1 go test -tags=integration ./internal/store/ -run TestMigratorSQLiteToPostgresE2E` |

> **警告**：`scripts/postgres-e2e-migrate.sh` 依赖 Docker 重置 schema。云上请使用**空库**，或在维护窗口内审批后 `DROP SCHEMA`，勿对生产库误跑。

---

## 4. RLS 安装与隔离（约 15 分钟）

```bash
export ASH_POSTGRES_RLS=1
export ASH_POSTGRES_RLS_FORCE=1
export ASH_DATABASE_APP_URL='postgres://ash_app:...@<rds-host>:5432/ash?sslmode=require'

go test -tags=integration ./internal/store/ \
  -run 'TestPostgresRLSPoliciesInstalled|TestPostgresRLSSpaceIsolationOnMemoryRecords' -count=1

export ASH_MIGRATE_E2E=1
go test -tags=integration ./internal/store/ -run TestPostgresRLSE2EAfterMigrate -count=1
```

| # | 检查 | 通过标准 |
|---|------|----------|
| 4.1 | 策略数量 | **M3-06** pass，`rlsPolicies` ≥ **41**（修订 000013–000020） |
| 4.2 | `ash_rls_tester` 隔离 | `TestPostgresRLSSpaceIsolationOnMemoryRecords` + `MemoryChildren` + `OrgIdentity` pass |
| 4.3 | `ash_app` 无 DDL | `TestPostgresRLSE2EAfterMigrate` pass |
| 4.4 | 跨 space 不可见 | 无 `ash_app` 泄漏行 |

---

## 5. Worker 运行时（约 10 分钟）

```bash
export ASH_DATABASE_APP_URL='...'
export ASH_POSTGRES_RLS=1
export ASH_POSTGRES_RLS_FORCE=1
export ASH_DATA_DIR='.ash'

go run ./cmd/worker &
curl -s http://localhost:8080/readyz

export ASH_DATABASE_APP_URL='...'
go run ./cmd/cli doctor --suite M3
```

| # | 检查 | 通过标准 |
|---|------|----------|
| 5.1 | `/readyz` | `dialect: postgres`，DB ping ok |
| 5.2 | **M3-07** | 设置 `ASH_DATABASE_APP_URL` → ping ok |
| 5.3 | 租户 API | 带 `X-ASH-Space-ID` 的 `GET /api/v1/runs` 仅本 space |
| 5.4 | 跨 space | 另一 space 资源 → `403 SPACE_ACCESS_DENIED` |
| 5.5 | Prometheus | `GET /metrics` 全局 scrape；`GET /api/v1/metrics/prometheus` 按 space |

---

## 6. Doctor 全量门禁（约 5–30 分钟）

```bash
export ASH_MIGRATE_E2E=1
export ASH_POSTGRES_RLS=1
export ASH_POSTGRES_RLS_FORCE=1
export ASH_DATABASE_APP_URL='...'

go run ./cmd/cli doctor --suite M3 --format md
go run ./cmd/cli doctor --suite ALL --agent static --format md
```

| 套件 | 项数 | 云 RDS 期望 |
|------|------|-------------|
| M3 | 8 | **全部 pass**（M3-04 / 06 / 07 / 08 非 skip） |
| ALL | 37 | **全部 pass**（含 M3-09 readyz 契约、TR3-05..07） |
| M3-05 ExecGo | 1 | 可选：`ASH_EXECGO_E2E=1` + live ExecGo；未启用可 skipped |

---

## 7. 业务抽样（约 15 分钟）

自动化（无需 UI）：

```bash
# 单测等价路径（CI / 无 Worker）
make release-sampling-static

# 对已启动 Worker（ASH_AUTH_MODE=dev 或 Bearer token）
export ASH_WORKER_URL=http://127.0.0.1:8080
ASH_CI_FIXTURE=1 make live-smoke   # H-04/05/06/07/09 live 编排
```

`make postgres-rds-e2e` 在设置 `ASH_WORKER_URL` 时自动调用 `live-smoke.sh`；否则跑 `make release-sampling-static`。烟测索引见 [`smoke-index.md`](smoke-index.md)。

| # | 场景 | 操作 | 期望 |
|---|------|------|------|
| 7.1 | 创建 Run | `POST /api/v1/runs` | 写入 RDS，`space_id` 正确 |
| 7.2 | SSE | `GET /runs/{id}/stream` | 事件流正常；审计 `stream.session_opened`（`TestReleaseSamplingSSE`） |
| 7.3 | Memory | 候选 → 评审 → query | 跨 space 读 → 403 |
| 7.4 | KPI | `GET /api/v1/metrics/overview?spaceId=...` | 与 SQLite 同期数据量级一致 |
| 7.5 | CI 诊断 | `POST /ci/failures/diagnose` | 落库 `ci_diagnoses`；Worker `ASH_CI_FIXTURE=1` 时用 `scripts/ci-fixture-smoke.sh` 或 `TestReleaseSamplingCIFixtureH04H05` |
| 7.6 | 合规导出 | `POST /compliance/export` | 含 audit + doctor 报告 |
| 7.7 | Scale 页 | `/ui/scale` | Postgres / RLS / app URL 状态正确 |

---

## 8. 切换与回滚

### 切换顺序（建议）

1. 维护窗口：`migrate copy` + `verify` 最终一次
2. 停 Worker（SQLite）
3. 设置 `ASH_DATABASE_APP_URL` + RLS 环境变量
4. 启 Worker（Postgres）
5. `readyz` + M3 + §7 抽样
6. 观察 30–60 分钟：`/metrics`、`observability/alerts`

### 回滚条件（任一即回滚）

- `migrate verify` 失败或抽样行数不一致
- M3-04 / M3-06 / M3-07 失败
- 跨 space 数据泄漏
- `readyz` 非 postgres 或持续 5xx
- P95 查询劣化超过约定 SLO

### 回滚动作

- Worker 改回 SQLite（unset `ASH_DATABASE_URL` / `ASH_DATABASE_APP_URL`）
- 保留 RDS 快照供复盘；**不要**在未验证前删除 SQLite

---

## 9. 与本地脚本的对应关系

| 本地 | 云 RDS 等价 |
|------|-------------|
| `make postgres-up` | RDS 实例 + 安全组 |
| `docker exec … DROP SCHEMA` | 空库或审批后 `psql` 重置 |
| `make postgres-e2e` | §1–§6 手工串联 |
| `make postgres-sql-schema-e2e` | §2 SQL-only（`ASH_SCHEMA_MODE=sql`） |
| `make postgres-roles` | `postgres-ensure-app-role.sh` + `psql` |
| `make postgres-rls-e2e` | §4 集成测试 + M3 RLS 环境 |

---

## 10. 证据归档（发布门禁）

- [ ] `migrate plan` / `verify` 输出日志
- [ ] `migrate schema version` 输出（version=20）
- [ ] `doctor --suite M3` 报告（11/11 pass）
- [ ] `doctor --suite ALL` 报告（43/43 pass）
- [ ] RLS 集成测试日志
- [ ] `readyz` 响应 + 切换时间戳
- [ ] §7 业务抽样记录

---

## 附录 A：快速串联（Git Bash / Linux）

已有 SQLite 与 RDS URL 时：

```bash
export ASH_DATABASE_URL='postgres://...'
export ASH_DATABASE_APP_URL='postgres://...'
export ASH_DATA_DIR='.ash'
export ASH_SQLITE_PATH="$ASH_DATA_DIR/ash.db"
export ASH_MIGRATE_E2E=1
export ASH_POSTGRES_RLS=1
export ASH_POSTGRES_RLS_FORCE=1

make postgres-rds-e2e
# 等价于 scripts/postgres-rds-e2e.sh
```

## 附录 B：相关文档

- `doc/05-M3-多租户与Postgres演进.md` — M3 架构与 RLS 说明
- `doc/TODO.md` — §1 Postgres E2E、§3 RLS 剩余项
- `scripts/postgres-init/01-ash-roles.sql` — `ash_app` / `ash_rls_tester` DDL
