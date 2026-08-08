# H-01～H-03 云环境验收与签字清单

> 在切换生产 `ASH_DATABASE_APP_URL` 前于目标云 RDS 执行。  
> 本地 Docker 等价：`make postgres-local-rds-e2e`（H-01 dry-run，含 M3-04）· `make postgres-app-gate`（H-02/H-03）。
>
> **一键云验收（证据归档）**：`make cloud-acceptance`（需 `ASH_DATABASE_URL`）  
> **MVP 发布签字**：`make mvp-signoff`（静态门禁 + 可选云/Docker）

## 0. 环境准备

| # | 项 | 负责人 | 验收 |
|---|-----|--------|------|
| 0.1 | RDS Postgres 13+，库 `ash` | 平台 | `bash scripts/postgres-smoke.sh` |
| 0.2 | 迁移账号 `ash`（DDL） | 平台 | 能 `migrate schema up` |
| 0.3 | 应用账号 `ash_app`（NOBYPASSRLS） | 平台 | `postgres-ensure-app-role.sh` |
| 0.4 | `sslmode=require` + 白名单 | 安全 | Worker/运维机可达 |
| 0.5 | 强密码已设置（非 dev 默认） | 安全 | 见 [`postgres-production-config.md`](postgres-production-config.md) |
| 0.6 | SQLite 快照 `.ash/ash.db` | 运维 | `migrate plan` 可对比 |

环境变量模板：[`config/cloud-rds.env.example`](../../config/cloud-rds.env.example)（复制为 `config/cloud-rds.env` 后 `source`）

```bash
cp config/cloud-rds.env.example config/cloud-rds.env
# 编辑 host / 密码
set -a && source config/cloud-rds.env && set +a

---

## H-01 云 RDS 全链路 E2E

| # | 检查 | 命令 | 通过标准 | 证据 |
|---|------|------|----------|------|
| H-01.1 | Schema 迁移 | `migrate schema up` + `version` | version=**20** expected=20 | `migrate-version.log` |
| H-01.2 | 数据迁移 | `migrate plan/copy/verify` | 全表行数一致 | `migrate-verify.log` |
| H-01.3 | Doctor M3 | `doctor --suite M3` | **11/11** pass，M3-04 非 skip | `doctor-m3.log` |
| H-01.4 | Doctor ALL | `doctor --suite ALL` | **43/43** pass | `doctor-all.log` |
| H-01.5 | TR3 live | `doctor --suite TR3` | TR3-06/10 pass | `doctor-tr3.log` |
| H-01.6 | 业务抽样 | `make release-sampling-static` 或 `live-smoke` | H-09 §7 | `release-sampling.log` |
| H-01.L | 本地 dry-run | `make postgres-local-rds-e2e` | Docker + sqlite→Postgres + M3-04 live | `postgres-local-rds-e2e.log` |

```bash
make cloud-acceptance
# 等价 scripts/cloud-acceptance-gate.sh → postgres-rds-e2e + 证据目录 .ash/evidence/cloud-h01-h03-*
# 无云时本地：make postgres-up && make postgres-local-rds-e2e
```

**H-01 签字**

| 角色 | 姓名 | 日期 | 备注 |
|------|------|------|------|
| 技术负责人 | | | |
| 测试负责人 | | | |

---

## H-02 云 RDS RLS + ash_app

| # | 检查 | 命令 | 通过标准 | 证据 |
|---|------|------|----------|------|
| H-02.1 | RLS 策略 | Doctor M3-06 | `rlsPolicies` ≥ **41** | `doctor-m3.log` |
| H-02.2 | ash_app ping | Doctor M3-07 | pass（需 `ASH_DATABASE_APP_URL`） | 同上 |
| H-02.3 | 租户隔离 | `go test -tags=integration ./internal/store/ -run TestPostgresRLS` | 全 pass | `rls-integration.log` |
| H-02.4 | 迁移后隔离 | `TestPostgresRLSE2EAfterMigrate` | pass | 同上 |
| H-02.5 | 本地等价 | `make postgres-app-gate` | Docker + schema 后 M3-06/07 | `postgres-app-gate.log` |

**H-02 签字**

| 角色 | 姓名 | 日期 | 备注 |
|------|------|------|------|
| 安全/平台 | | | |
| 测试负责人 | | | |

---

## H-03 生产 Worker 配置

| # | 检查 | 操作 | 通过标准 | 证据 |
|---|------|------|----------|------|
| H-03.1 | 运行时连接 | Worker 设 `ASH_DATABASE_APP_URL` | store 使用 `ash_app` | env 截图/清单 |
| H-03.2 | RLS 强制 | `ASH_POSTGRES_RLS=1` + `FORCE=1` | 无 bypass | env 清单 |
| H-03.3 | Schema 模式 | `ASH_SCHEMA_MODE=sql` | AutoMigrate 关闭 | `/readyz` JSON |
| H-03.4 | readyz | `curl /readyz` | `dialect=postgres`，`sqlMigrationVersion=20` | `readyz.json` |
| H-03.5 | 租户 API | `GET /runs` + `X-ASH-Space-ID` | 仅本 space | 抽样记录 |
| H-03.6 | 跨 space | 访问他 space 资源 | `403 SPACE_ACCESS_DENIED` | 抽样记录 |

```bash
# 启动 Worker（生产 env）
export ASH_DATABASE_APP_URL='...'
export ASH_POSTGRES_RLS=1
export ASH_POSTGRES_RLS_FORCE=1
export ASH_SCHEMA_MODE=sql
go run ./cmd/worker

# 另一终端
curl -s http://<worker>/readyz | tee readyz.json
go run ./cmd/cli doctor --suite M3 --require M3-07
ASH_WORKER_URL=http://<worker> ASH_CI_FIXTURE=1 make live-smoke
```

**H-03 签字**

| 角色 | 姓名 | 日期 | 备注 |
|------|------|------|------|
| 发布负责人 | | | |
| 值班 SRE | | | |

---

## 证据归档

| 产物 | 路径 | 入库 |
|------|------|------|
| 原始日志 | `.ash/evidence/cloud-h01-h03-*` | 否（gitignore） |
| 云验收摘要 | `doc/evidence/cloud-h01-h03-latest.md` | 是（签字后提交） |
| MVP 签字报告 | `doc/evidence/mvp-signoff-YYYY-MM-DD.md` | 是 |

---

## 与 MVP 清单的对应

| MVP § | 项 | 本清单 |
|-------|-----|--------|
| 5 | Postgres 生产切换 | H-01 + [`release-window-audit.md`](release-window-audit.md) |
| 5 | H-04~H-09 | H-01 §7 / `live-smoke` |
| 6 | staging 迁移 | H-01.1~H-01.2 |
| 10 | T+0 冒烟 | H-03.6 + `live-smoke` |

相关：[`postgres-rds-e2e.md`](postgres-rds-e2e.md) · [`postgres-app-gate.md`](postgres-app-gate.md) · [`smoke-index.md`](smoke-index.md)
