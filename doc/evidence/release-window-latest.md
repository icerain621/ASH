# 发布窗口记录（2026-07-07）

- 生成时间（UTC）：2026-07-07T13:50:00Z
- Git：9737502 @ main
- 手册：[`release-window-runbook.md`](../checklists/release-window-runbook.md)

## 门禁快照

| 门禁 | 命令 | 状态 |
|------|------|------|
| 配置核对 | `make config-env-gate` | ✅（prior） |
| T+0 告警 | `make t0-alert-gate` | ✅（prior） |
| T+1 指标 | `make t1-metrics-gate` | ✅（prior） |
| KPI §9 对账 | `make kpi-reconcile-gate` | ✅ 2026-07-07 |
| 数据备份 | `make data-backup` | ✅（prior） |
| 迁移前 | `make pre-migrate-gate` | ✅（cloud-acceptance 内） |
| 云 RDS | `make cloud-acceptance` | ✅ 本地 Docker（2026-07-07）；云 RDS 待环境 |
| Postgres app | `make postgres-app-gate` | ✅ 本地 H-03 readyz（2026-07-07） |
| Live Worker | `make worker-local-gate` | ⏸ |
| MVP 签字 | `make mvp-signoff` | ✅（占位 dry-run） |

## 切换检查

- [x] `make data-backup`（prior）
- [x] `make pre-migrate-gate`（cloud-acceptance 2026-07-07）
- [x] `make cloud-acceptance`（**本地 Docker**；云 RDS 待 `cloud-rds.env`）
- [x] `make postgres-app-gate`（本地 H-03）
- [x] `make kpi-reconcile-gate`
- [ ] `make t0-alert-gate`（切换日再跑）
- [ ] `make t1-metrics-gate`（T+1 生产数据）

## 值班 roster（填写）

| 角色 | 姓名 | 联系方式 |
|------|------|----------|
| 发布负责人 | | |
| 技术 on-call | | |
| DBA / 平台 | | |

## 待人工

- §8 发布窗口时间与值班表
- §11 四人签字：[`mvp-signoff-latest.md`](mvp-signoff-latest.md)
- 云 RDS 正式验收：[`h01-h03-cloud-signoff.md`](../checklists/h01-h03-cloud-signoff.md)
