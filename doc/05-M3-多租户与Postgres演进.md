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
- Postgres **RLS**（行级安全）作为第二道防线（P2）
- 跨空间管理操作显式 `org:admin` 权限 + 审计（P2）

## 3. Postgres 迁移路径

### 3.1 配置

```bash
# 开发（默认）
unset ASH_DATABASE_URL
# 数据目录下 ash.db（glebarez/sqlite，无需 CGO）

# 迁移验证目标（仅供 Doctor M3-04 / e2e 使用）
export ASH_MIGRATE_POSTGRES_URL='postgres://ash:ash@127.0.0.1:5432/ash?sslmode=disable'

# 生产
export ASH_DATABASE_URL='postgres://ash:secret@db.example.com:5432/ash?sslmode=require'
```

支持 URL 形式：`postgres://`、`postgresql://`、libpq 关键字 DSN（`host=... dbname=...`）。
`ASH_MIGRATE_POSTGRES_URL` 表示 SQLite→Postgres 迁移验证目标；`ASH_DATABASE_URL` 表示当前 ASH 主库，切换生产主库时才设置。

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
7. **回归**：`doctor --suite ALL --agent static`；TR0 回放一致性不得回退。若 ExecGo 执行面已就绪，可追加默认 agent 的 ALL 套件。

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

Doctor **M3-02** 校验 Postgres URL 解析与当前方言一致性；**M3-03** 校验 `ash migrate` 表目录与当前 schema 一致；**M3-04** 在 `ASH_MIGRATE_E2E=1` 且 `ASH_MIGRATE_POSTGRES_URL` 指向 live Postgres 时执行 `migrate verify`（默认跳过）。

### 3.0 本地 Postgres（Docker）

```bash
make postgres-up          # 启动 postgres:16，并打印可复制的 ASH_DATABASE_URL
make postgres-e2e         # 重置 disposable postgres，执行 plan/copy/verify + Doctor M3-04 + readyz + ALL 回归
make postgres-down        # 停止并清理卷
```

## 5. 验证命令

```bash
go run ./cmd/cli doctor --suite M3
go run ./cmd/cli doctor --suite M2   # 含 M2-03 运行期 POLICY_DENIED
bash scripts/postgres-smoke.sh
bash scripts/migrate-postgres.sh plan
go run ./cmd/cli migrate dual-write status --postgres "$ASH_DATABASE_URL"
```
