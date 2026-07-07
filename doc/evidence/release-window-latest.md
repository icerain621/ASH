# 发布窗口记录（2026-07-07）

- 生成时间（UTC）：2026-07-07T14:21:00Z
- Git：ebc3bcc @ main
- 手册：[`release-window-runbook.md`](../checklists/release-window-runbook.md)

## 门禁快照

| 门禁 | 命令 | 状态 |
|------|------|------|
| 配置核对 | `make config-env-gate` | ✅（prior） |
| T+0 告警 | `make t0-alert-gate` | ✅ 2026-07-07（BI-5 切换日演练） |
| T+1 指标 | `make t1-metrics-gate` | ✅（prior） |
| KPI §9 对账 | `make kpi-reconcile-gate` | ✅ 2026-07-07 |
| 数据备份 | `make data-backup` | ✅ 2026-07-07（`.ash/backups/ash-20260707T141951Z.db`） |
| 迁移前 | `make pre-migrate-gate` | ✅（cloud-acceptance 内） |
| 云 RDS | `make cloud-acceptance` | ✅ 本地 Docker（2026-07-07）；云 RDS 待环境 |
| Postgres app | `make postgres-app-gate` | ✅ 本地 H-03 readyz（2026-07-07） |
| Live Worker | `make worker-local-gate` | ✅ 2026-07-07（H-04~H-09 live） |
| MVP 签字 | `make signoff-gate` | ✅ 2026-07-07（占位 dry-run） |

## 切换检查（BI-5 本地演练）

- [x] `make data-backup`（2026-07-07T14:19:51Z）
- [x] 停 SQLite Worker（无进程）
- [x] migrate schema rev 20 + `copy` + `verify`（14560 行，全表 match）
- [x] `ASH_DATABASE_APP_URL` + RLS env 已设（5433）
- [x] `make worker-local-gate`（2026-07-07；H-04~H-09 live + SSE）
- [x] `make t0-alert-gate`（2026-07-07）
- [ ] `make t1-metrics-gate`（T+1 生产数据）

## 值班 roster（填写）

| 角色 | 姓名 | 联系方式 |
|------|------|----------|
| 发布负责人 | 发布负责人（占位） | |
| 技术 on-call | 技术 on-call（占位） | |
| DBA / 平台 | DBA（占位） | |

## 待人工

- §8 发布窗口 roster：[`release-window-runbook.md`](../checklists/release-window-runbook.md)（占位 dry-run ✅；联系方式待填）
- §11 四人签字：[`mvp-signatures-latest.md`](mvp-signatures-latest.md)（占位 dry-run ✅）
- 云 RDS 正式验收：[`h01-h03-cloud-signoff.md`](../checklists/h01-h03-cloud-signoff.md)
