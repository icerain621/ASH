# H-01~H-03 云环境验收证据

- 生成时间（UTC）：2026-07-06T15:17:55Z
- Git：91f9015 @ main
- 证据目录：`/c/Go_Work/src/ash/.ash/evidence/cloud-h01-h03-20260706T151646Z`（本地，未入库）
- RDS host：`postgres://ash:ash@...`（脱敏）

## 验收项

| 项 | 状态 | 说明 |
|----|------|------|
| H-01 云 RDS 全链路 | ✅ | `postgres-rds-e2e` + migrate verify + Doctor ALL |
| H-02 RLS + ash_app | ✅ | M3-06/07 + RLS 集成测试 |
| H-03 生产 Worker 配置 | ⏸ | 需 Worker 指向 `ASH_DATABASE_APP_URL` 后 §5 readyz 签字 |

## 签字（云验收完成后填写）

| 角色 | 姓名 | 日期 |
|------|------|------|
| 技术负责人 | | |
| 测试负责人 | | |
| 发布负责人 | | |

清单：[`h01-h03-cloud-signoff.md`](../checklists/h01-h03-cloud-signoff.md)
