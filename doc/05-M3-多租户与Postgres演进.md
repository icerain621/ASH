# M3：多租户强隔离与 Postgres 演进 v0.1

> 文档状态：展望 + 可执行检查清单。配合 Doctor `M3` 套件与 `/api/v1/scale/readiness` 数据库字段。

## 1. 目标

- **多租户强隔离**：所有租户数据以 `space_id` 为边界；跨空间访问默认拒绝并审计。
- **存储演进**：开发/单机继续 SQLite（纯 Go 驱动）；生产设置 `ASH_DATABASE_URL` 切换 Postgres，无需改业务模型。

## 2. 多租户隔离（已实现基础）

| 机制 | 说明 |
|------|------|
| `store.EnforceSpaceAccess` | API 层校验记录 `space_id` 与请求上下文一致 |
| `store.SpaceWhere` | GORM 查询统一按 `space_id` 过滤 |
| 记忆读取 | `GET /memory/records/:id` 跨空间返回 `403 SPACE_ACCESS_DENIED` |
| Doctor M3-01 | 双空间记忆写入 + 跨空间查询不得泄漏 |

### 2.1 后续强化

- ✅ **API 租户边界**（v0.2）：`requireRequestSpace` / `requireTargetSpace`；`spaceForParam`、runs/secrets/RAG/MCP/审批等路径统一 `EnforceSpaceAccess` → `403 SPACE_ACCESS_DENIED`
- Postgres **RLS**（行级安全）骨架（P2，可选启用）：`ASH_POSTGRES_RLS=1` 安装 `ash_space_*` 策略；`ASH_POSTGRES_RLS_FORCE=1` 对表 owner 也生效；迁移路径使用 `app.ash_rls_bypass=on`
- 跨空间管理操作显式 `org:admin` 权限 + 审计（P2）

## 3. Postgres 迁移路径

### 3.1 配置

```bash
# 开发（默认）
unset ASH_DATABASE_URL
# 数据目录下 ash.db（glebarez/sqlite，无需 CGO）

# 生产
export ASH_DATABASE_URL='postgres://ash:secret@db.example.com:5432/ash?sslmode=require'
```

支持 URL 形式：`postgres://`、`postgresql://`、libpq 关键字 DSN（`host=... dbname=...`）。

### 3.2 迁移步骤（建议）

1. **准备**：部署 Postgres 13+；创建库与用户；配置备份与连接池（PgBouncer 可选）。
2. **冒烟**：`bash scripts/postgres-smoke.sh`（解析 URL + 可选 live 连接）。
3. **Schema**：使用当前 GORM `AutoMigrate` 在新库建表（M0 策略）；P2 引入 `golang-migrate` 版本化。
4. **数据迁移**（停机或双写窗口）— 使用 `ash migrate` CLI：
   ```bash
   # 行数对比（源 sqlite / 目标 postgres）
   go run ./cmd/cli migrate plan --postgres "$ASH_DATABASE_URL"

   # 全量 upsert（GORM OnConflict，按主键覆盖）
   go run ./cmd/cli migrate copy --postgres "$ASH_DATABASE_URL"

   # 行数校验（不一致则非零退出）
   go run ./cmd/cli migrate verify --postgres "$ASH_DATABASE_URL"

   # 增量同步（检查点：.ash/migration/sync-state.json）
   go run ./cmd/cli migrate sync --postgres "$ASH_DATABASE_URL"

   # 或封装脚本
   export ASH_DATABASE_URL='postgres://ash:ash@127.0.0.1:5432/ash?sslmode=disable'
   bash scripts/migrate-postgres.sh plan
   bash scripts/migrate-postgres.sh copy
   bash scripts/migrate-postgres.sh verify
   ```
5. **双写窗口**（可选，sqlite 主库 + postgres 影子）：
   ```bash
   go run ./cmd/cli migrate dual-write enable --postgres "$ASH_DATABASE_URL"
   # 重启 Worker（sqlite 主库）后自动从 dual-write.json 加载影子 Postgres
   # 可选覆盖：export ASH_DUAL_WRITE_POSTGRES_URL="$ASH_DATABASE_URL"
   go run ./cmd/cli migrate dual-write sync --postgres "$ASH_DATABASE_URL"
   go run ./cmd/cli migrate dual-write disable
   ```
   状态文件：`.ash/migration/dual-write.json`；Worker 优先读 `ASH_DUAL_WRITE_POSTGRES_URL`，否则读已启用的配置文件。
6. **切换**：设置 `ASH_DATABASE_URL`，重启 Worker；`go run ./cmd/cli doctor --suite M3` 确认 `M3-02` 通过。
7. **回归**：`doctor --suite ALL`；TR0 回放一致性不得回退。

### 3.3 触发迁移的信号（摘自架构文档）

- SQLite 写锁 / 备份窗口不可接受
- `run_events` / `audit_log` 体量导致查询 P95 劣化
- 多实例 Worker 并发写冲突
- 组织级合规要求独立 DB 集群

## 4. 可观测与就绪

`GET /api/v1/scale/readiness` 返回：

- `databaseDialect`：`sqlite` | `postgres`
- `postgresConfigured`：是否通过 `ASH_DATABASE_URL` 使用 Postgres
- `migrationReady`：当前方言是否纳入支持矩阵
- `postgresRLSEnabled` / `postgresRLSForce`：Postgres 行级安全骨架是否启用

### 2.2 Postgres RLS 骨架（第二道防线）

```bash
export ASH_DATABASE_URL='postgres://ash:ash@127.0.0.1:5432/ash?sslmode=disable'
export ASH_POSTGRES_RLS=1          # 启动时安装 ash_space_* 策略 + GORM 回调
export ASH_POSTGRES_RLS_FORCE=1    # 可选：表 owner 也受策略约束（集成测试 / 生产 ash_app 角色）
make postgres-roles                # 创建 ash_app / ash_rls_tester（scripts/postgres-init/01-ash-roles.sql）
export ASH_DATABASE_APP_URL='postgres://ash_app:ash_app@127.0.0.1:5432/ash?sslmode=disable'  # Worker 运行时
make postgres-rls-e2e              # RLS 策略 + ash_rls_tester 隔离 + 可选 migrated-db ash_app 探针
# 迁移/Doctor 仍用 ASH_DATABASE_URL（owner）；Worker store.Open 优先 ASH_DATABASE_APP_URL
# Doctor M3-07 在设置 ASH_DATABASE_APP_URL 时 ping ash_app 连接
```

- 会话变量：`app.ash_space_id`（租户空间）、`app.ash_rls_bypass=on`（迁移/管理）
- org 管理员 `GET /orgs`、`GET /spaces` 可走 RLS bypass（需 `org:write`，审计 `tenant.rls_bypass`）
- API：`rlsMiddleware` 将认证空间写入 `request.Context`；查询请使用 `h.dbFor(c)`（`WithContext`）
- 程序化：`db.TransactionWithRLSSpace(space, fn)` / `db.TransactionWithRLSBypass(fn)`
- 覆盖表：含 `space_id` 的业务表 + `run_*` 子表（经 `runs.space_id` 关联）
- 验证：`go test -tags=integration ./internal/store/ -run TestPostgresRLS -count=1`（需 Postgres）

### 2.3 指标与 Prometheus（v1 策略）

| 端点 | 认证 | 租户范围 | RLS 行为 |
|------|------|----------|----------|
| `GET /metrics` | 无（运维 scrape） | **全局**（所有 space 聚合） | `ASH_POSTGRES_RLS=1` 时使用 `app.ash_rls_bypass` 读取全库 |
| `GET /api/v1/metrics/prometheus` | 是 | 当前会话 space（可 `?spaceId=`） | 走 `rlsMiddleware` + SQL `space_id` 过滤；序列带 `space_id` 标签 |
| `GET /api/v1/metrics/overview` | 是 | 同上 | KPI 聚合；`OverviewContext` 绑定请求 context |

- 控制台 `/ui/observability` 使用 **租户** `/api/v1/metrics/prometheus`；`/ui/metrics` 使用 overview API。
- 生产 Prometheus 继续 scrape `GET /metrics`；按空间看板请用 overview 或 tenant prometheus API。

Doctor **M3-02** 校验 Postgres URL 解析与当前方言一致性；**M3-03** 校验 `ash migrate` 表目录与当前 schema 一致；**M3-04** 在 `ASH_MIGRATE_E2E=1` 时对 live Postgres 执行 `migrate verify`（默认跳过）。

### 3.0 本地 Postgres（Docker）

```bash
make postgres-up          # 启动 postgres:16，导出 ASH_DATABASE_URL
make postgres-e2e         # 完整 sqlite→postgres plan/copy/verify + doctor M3
make postgres-down        # 停止并清理卷
```

### 3.4 PgBouncer / 连接池（生产建议）

| 角色 | 连接目标 | 说明 |
|------|----------|------|
| Worker（`ash_app`） | PgBouncer **transaction** 模式 → RDS | `ASH_DATABASE_APP_URL` 指向 pooler；RLS session 变量在事务内有效 |
| 迁移 / Doctor（owner） | 直连 RDS 或 pooler **session** 模式 | `ASH_DATABASE_URL`；需 DDL / `ALTER` RLS |
| Prometheus scrape | 无 DB 或全局 bypass | `GET /metrics` 不经过 pooler 租户会话 |

建议：

- Pool size：按 Worker 实例数 × 并发 × 1.2 预留；`ash_app` 无 DDL 权限。
- `server_reset_query = DISCARD ALL`（transaction pooling）避免 RLS 会话变量泄漏。
- `/ui/scale` 就绪卡可查看 `workerConnectionRole`、`runtimeDsnHint`、双写影子库 URL（脱敏）。

## 5. 云 RDS E2E 清单

生产切换前在目标 RDS 执行完整验证，见 **[`doc/checklists/postgres-rds-e2e.md`](checklists/postgres-rds-e2e.md)**（迁移、RLS、`ash_app`、Doctor M3/ALL、业务抽样与回滚）。

## 6. 验证命令

```bash
go run ./cmd/cli doctor --suite M3
go run ./cmd/cli doctor --suite M2   # 含 M2-03 运行期 POLICY_DENIED
bash scripts/postgres-smoke.sh
bash scripts/migrate-postgres.sh plan
go run ./cmd/cli migrate dual-write status --postgres "$ASH_DATABASE_URL"
```
