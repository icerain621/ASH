# ASH 待办 / 技术债

> 记录尚未完成或需人工环境验证的项。完成请移入 CHANGELOG 并删除对应条目。

## 1. Postgres 端到端迁移验证

**状态**：已接入一键验证（Docker，生产切换前仍需在目标环境复跑）
**优先级**：P1（切换生产前必做）

本地一键验证（需 Docker）：

```bash
make postgres-e2e

# 仅启动本地 Postgres，并按输出提示在当前 shell 中设置 ASH_DATABASE_URL：
make postgres-up
export ASH_DATABASE_URL='postgres://ash:ash@127.0.0.1:5432/ash?sslmode=disable'
make postgres-down
```

集成测试（Postgres 已启动且 `ASH_DATABASE_URL` 已设置）：

```bash
export ASH_DATABASE_URL='postgres://ash:ash@127.0.0.1:5432/ash?sslmode=disable'
export ASH_MIGRATE_E2E=1
make test-integration
```

**验收标准**：

- `migrate verify` 全表行数一致
- Doctor **M3-04** 在 `ASH_MIGRATE_E2E=1` 且 `ASH_MIGRATE_POSTGRES_URL` 指向 live Postgres 时通过（默认跳过）
- 切换 `ASH_DATABASE_URL` 后 Worker `readyz` 通过
- `doctor --suite ALL --agent static` 无回退

**备注**：`ASH_MIGRATE_POSTGRES_URL` 只用于 SQLite→Postgres 迁移验证目标；`ASH_DATABASE_URL` 表示 Worker/CLI 当前主库。云 RDS / 生产切换前仍需在目标环境重复 `copy` + `verify` + `/readyz` + 抽样业务校验。

---

## 已完成（近期）

- M3 API 租户隔离：`requireRequestSpace` / `requireTargetSpace` + `spaceForParam` 强制校验
- Worker 启动自动读取 `.ash/migration/dual-write.json`（`ASH_DUAL_WRITE_POSTGRES_URL` 优先）
- `docker-compose.postgres.yml` + `scripts/postgres-{up,down,e2e-migrate}.sh`
- Doctor M3-04 使用 `ASH_MIGRATE_POSTGRES_URL` 验证 live Postgres 迁移目标，避免和主库 `ASH_DATABASE_URL` 混用
