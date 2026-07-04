# Postgres 新表 RLS 接入清单

> 新增 `CREATE TABLE` 修订后，按表类型同步 Go catalog、SQL RLS 扩展与 Doctor **M3-11** 校验。

## 1. 分类决策

| 表特征 | 分类 | 接入位置 |
|--------|------|----------|
| 有 `space_id`（或 `spaces.id`） | 租户 | `PostgresRLSTables()` + `000013` VALUES 块 |
| 仅有 `run_id` | run-scoped | `postgresRLSRunScopedTables()` + `000018+` |
| 仅有 `memory_id` | memory-scoped | `postgresRLSMemoryScopedTables()` + `000019+` |
| `users`/`orgs`/`roles`/`members` 模式 | org-scoped | `PostgresRLSOrgScopedTables()` + `000020+` |
| 无租户列、部署全局 | global | `PostgresRLSGlobalTables()`（**不**进 000013） |

`MigrationCatalogRLSGaps()` 必须为空；`VerifyRLSMigrationSQL()` 必须通过。

## 2. 工作流

1. 追加 SQL 修订 `0000NN_*.up.sql`（表 DDL + 可选 RLS 扩展 DO 块）
2. 更新 `internal/store/sqlmigrations/migrate.go` `expectedVersion`
3. 更新 `internal/store/rls.go` / `rls_catalog.go` 目录函数
4. 更新 `internal/store/rls_migration_test.go` 策略计数
5. 若为新策略类型：扩展 `ApplyPostgresRLSPolicies()` 运行时 backfill
6. Postgres 集成测试 + `make postgres-rls-e2e`
7. `go test ./internal/doctor/... -run TestM3Suite`（**M3-11**）

## 3. 策略命名

- 统一前缀：`ash_space_<table>`
- Session 变量：`app.ash_space_id`、`app.ash_org_id`、`app.ash_rls_bypass`
- Worker middleware：`internal/api/rls_middleware.go` 解析 `spaces.org_id`

## 4. 验收

```bash
go test ./internal/store -run 'TestMigrationCatalog_RLSCoverage|TestVerifyRLSMigrationSQL|TestRLSExpectedPolicyCount' -count=1
make postgres-rls-e2e   # 需 Docker Postgres
go run ./cmd/cli doctor --suite M3
```

`GET /api/v1/scale/readiness` 应暴露 `postgresRLSPolicyCount` / `postgresRLSPolicyExpected`（当前 **41**）。
