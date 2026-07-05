# 发布窗口记录（2026-07-05）

- 生成时间（UTC）：2026-07-05T13:42:31Z
- Git：a56fc1b @ main
- 门禁命令：`make release-window-gate`
- 自动化步骤通过：**10**
- 证据目录：`/c/Go_Work/src/ash/.ash/evidence/release-window-20260705T134044Z`（本地，未入库）
- 手册：[`release-window-runbook.md`](../checklists/release-window-runbook.md)

## 门禁快照

| 门禁 | 命令 | 状态 |
|------|------|------|
| 配置核对 | `make config-env-gate` | ✅ |
| T+0 告警 | `make t0-alert-gate` | ✅ |
| T+1 指标 | `make t1-metrics-gate` | ✅ |
| 数据备份 | `make data-backup` | ✅ |
| 迁移前 | `make pre-migrate-gate` | ✅ |
| 回滚演练 | `make rollback-drill` | ✅ |
| MVP 签字 | `make mvp-signoff` | ✅（prior mvp-signoff） |
| Live Worker | `make worker-local-gate` | ⏸ |
| JWT Worker | `make worker-production-gate` | ⏸ |
| 云 RDS | `make cloud-acceptance` | ⏸ 待环境 |

## 切换检查

- [x] `make data-backup`
- [x] `make pre-migrate-gate`
- [ ] `make cloud-acceptance`（云环境）
- [x] `make t0-alert-gate`
- [x] `make t1-metrics-gate`（T+1）

## 值班 roster（填写）

| 角色 | 姓名 | 联系方式 |
|------|------|----------|
| 发布负责人 | | |
| 技术 on-call | | |
| DBA / 平台 | | |

## 待人工

- 发布窗口时间与值班表（§8 roster）
- §11 四人签字：[`mvp-signoff-latest.md`](mvp-signoff-latest.md)
- 云 RDS：[`h01-h03-cloud-signoff.md`](../checklists/h01-h03-cloud-signoff.md)
