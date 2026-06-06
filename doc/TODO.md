# ASH 待办 / 技术债

> 记录尚未完成或需人工环境验证的项。完成请移入 CHANGELOG 并删除对应条目。

## 1. Postgres 端到端迁移验证

**状态**：可执行（Docker）  
**优先级**：P1（切换生产前必做）

本地一键验证（需 Docker）：

```bash
make postgres-e2e
# 或分步：
make postgres-up
export ASH_DATABASE_URL='postgres://ash:ash@127.0.0.1:5432/ash?sslmode=disable'
bash scripts/postgres-e2e-migrate.sh
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
- Doctor **M3-04** 在 `ASH_MIGRATE_E2E=1` 时通过（默认跳过）
- 切换 `ASH_DATABASE_URL` 后 Worker `readyz` 通过
- `doctor --suite ALL` 无回退

**备注**：云 RDS / 生产切换前仍需在目标环境重复 `copy` + `verify` + 抽样业务校验。

---

## 2. CI / ExecGo / Repo 诊断 / KPI 验证

**状态**：已实现 MVP，本地与 CI 需持续回归
**优先级**：P0（PRD 前四项闭环）

入口：

```bash
go test ./internal/ci ./internal/metrics ./internal/api -run 'TestDiagnose|TestOverview|TestCreateRepo|TestRepoConnection' -count=1
make execgo-health
ASH_EXECGO_E2E=1 go run ./cmd/cli doctor --suite M3 --format md --agent execgo_codex
go run ./cmd/cli doctor --suite ALL --format md --agent static
make web-build
```

**验收标准**：

- GitHub Actions PR/main 门禁执行 Go、Doctor static 与前端 build
- Postgres e2e 仅在 nightly/manual workflow 执行，避免拖慢普通 PR
- ExecGo live smoke 在 `ASH_EXECGO_E2E=1` 时失败可定位，未启用时 M3-05 明确 skipped
- Repo connection 只接受 `secretId`，拒绝明文 token
- `POST /api/v1/ci/failures/diagnose` 可落库并输出 rootCause / fixSuggestions / evidenceRefs
- `/ui/metrics` 与 `GET /api/v1/metrics/overview` 展示 KPI v1；缺失 SSE 数据源时返回 unavailable，不造假

**剩余人工项**：

- 在真实 GitHub token 下验证 `GET /api/v1/ci/runs?sync=true`
- 在 live ExecGo / execgo-runtime 环境下验证 M3-05 通过
- 将 repo connection 的 token secret 接入团队密钥轮换策略

---

## 已完成（近期）

- M3 API 租户隔离：`requireRequestSpace` / `requireTargetSpace` + `spaceForParam` 强制校验
- Worker 启动自动读取 `.ash/migration/dual-write.json`（`ASH_DUAL_WRITE_POSTGRES_URL` 优先）
- `docker-compose.postgres.yml` + `scripts/postgres-{up,down,e2e-migrate}.sh`
- GitHub Actions CI 门禁 + manual/nightly Postgres e2e workflow
- ExecGo health 分类输出 + Doctor M3-05 live smoke
- GitHub repo connection、CI run 摘要、CI 失败确定性诊断 API
- KPI overview API 与 `/ui/metrics` 控制台页面
