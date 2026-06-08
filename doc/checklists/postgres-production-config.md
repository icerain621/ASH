# Postgres 生产配置模板（revision 15）

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
# 期望：version=14 dirty=false expected=14 mode=sql

# 2. SQLite → Postgres 数据（若有存量）
export ASH_MIGRATE_E2E=1
go run ./cmd/cli migrate plan  --data-dir "$ASH_DATA_DIR" --sqlite "$ASH_SQLITE_PATH" --postgres "$ASH_DATABASE_URL"
go run ./cmd/cli migrate copy  --data-dir "$ASH_DATA_DIR" --sqlite "$ASH_SQLITE_PATH" --postgres "$ASH_DATABASE_URL"
go run ./cmd/cli migrate verify --data-dir "$ASH_DATA_DIR" --sqlite "$ASH_SQLITE_PATH" --postgres "$ASH_DATABASE_URL"

# 3. 门禁
go run ./cmd/cli doctor --suite M3 --format md
go run ./cmd/cli doctor --suite ALL --agent static --format md
# 期望：M3 8/8，ALL 35/35（M3-08 sqlVersion=17，TR3-05 metricsParity，TR3-06 postgres FTS）
```

## Worker 启动检查

```bash
curl -s http://<worker>/readyz | jq .
# dialect=postgres；Scale readiness：schemaMode=sql、sqlMigrationVersion=14、autoMigrateEnabled=false
```

## 双写

`ASH_SCHEMA_MODE=sql` 生产路径不应再依赖 `ASH_DUAL_WRITE_POSTGRES_URL`。若仍配置，Scale readiness 应显式关闭双写并归档切换证据。

## 相关清单

- [`postgres-rds-e2e.md`](postgres-rds-e2e.md) — 云 RDS 全链路
- [`../05-M3-多租户与Postgres演进.md`](../05-M3-多租户与Postgres演进.md) — 架构说明
