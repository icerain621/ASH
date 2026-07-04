# Postgres 生产配置模板（revision 20）

> 切换生产前复制本模板为环境变量/密钥配置。本地验证：`make postgres-sql-schema-e2e`、`make postgres-rls-e2e`。

## 推荐 Schema 模式

生产试点建议使用 **SQL-only** 建表，关闭 GORM AutoMigrate：

```bash
export ASH_SCHEMA_MODE=sql
# 等价：export ASH_DISABLE_AUTOMIGRATE=1
```

## 连接与角色

```bash
# 迁移 / Doctor / DDL（owner，可 bypass RLS 安装策略）
export ASH_DATABASE_URL='postgres://ash:<MIGRATOR_PW>@<host>:5432/ash?sslmode=require'

# Worker 运行时（ash_app，NOBYPASSRLS）
export ASH_DATABASE_APP_URL='postgres://ash_app:<APP_PW>@<host>:5432/ash?sslmode=require'

export ASH_DATA_DIR='.ash'
export ASH_SQLITE_PATH="$ASH_DATA_DIR/ash.db"
```

## RLS

```bash
export ASH_POSTGRES_RLS=1
export ASH_POSTGRES_RLS_FORCE=1
```

策略由 SQL 修订 **000013** 安装；**000014** 授予 `ash_app` DML。Worker 连接 `ash_app` 时运行时仅注册 session 回调与可选 `FORCE`。

## 迁移顺序（空库 / 维护窗口）

```bash
# 1. 应用全部 SQL 修订（000001–000017，expectedVersion=17）
go run ./cmd/cli migrate schema up --postgres "$ASH_DATABASE_URL"
go run ./cmd/cli migrate schema version --postgres "$ASH_DATABASE_URL"
# 期望：version=17 dirty=false expected=17 mode=sql

# 2. SQLite → Postgres 数据（若有存量）
export ASH_MIGRATE_E2E=1
go run ./cmd/cli migrate plan  --data-dir "$ASH_DATA_DIR" --sqlite "$ASH_SQLITE_PATH" --postgres "$ASH_DATABASE_URL"
go run ./cmd/cli migrate copy  --data-dir "$ASH_DATA_DIR" --sqlite "$ASH_SQLITE_PATH" --postgres "$ASH_DATABASE_URL"
go run ./cmd/cli migrate verify --data-dir "$ASH_DATA_DIR" --sqlite "$ASH_SQLITE_PATH" --postgres "$ASH_DATABASE_URL"

# 3. 门禁
go run ./cmd/cli doctor --suite M3 --format md
go run ./cmd/cli doctor --suite ALL --agent static --format md
# 期望：M3 11/11，ALL 41/41（M3-09 readyz 契约，M3-10 RLS 全局表，M3-11 RLS 目录，TR3-05..09）
```

## 可观测性（可选）

```bash
# Prometheus /metrics 追加 run_events 离线 replay 段（与 derive catalog 口径一致）
export ASH_METRICS_EVENT_REPLAY=1

# 后台治理告警评估（最短 1m；未设置则仅手动 POST /observability/alerts/evaluate）
export ASH_ALERTS_EVAL_INTERVAL=5m

# OTel traces（需可达 OTLP collector）
export ASH_OTEL_ENABLED=1
export ASH_OTEL_ENDPOINT=otel-collector:4317
```

配置说明见 `config/ash-observability.yaml`。

## Worker 启动检查

```bash
curl -s http://<worker>/readyz | jq .
# dialect=postgres
# schemaMode=sql、sqlMigrationVersion=17、autoMigrateEnabled=false
# otelEnabled、alertsEvalInterval、metricsEventReplayEnabled 与上表环境变量一致
```

## 双写

`ASH_SCHEMA_MODE=sql` 生产路径不应再依赖 `ASH_DUAL_WRITE_POSTGRES_URL`。若仍配置，Scale readiness 应显式关闭双写并归档切换证据。

## 相关清单

- [`postgres-rds-e2e.md`](postgres-rds-e2e.md) — 云 RDS 全链路
- [`../05-M3-多租户与Postgres演进.md`](../05-M3-多租户与Postgres演进.md) — 架构说明
