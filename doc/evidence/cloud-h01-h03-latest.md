# H-01~H-03 云环境验收证据

- 生成时间（UTC）：2026-07-07T13:47:03Z
- Git：9737502 @ main
- 证据目录：`/c/Go_Work/src/ash/.ash/evidence/cloud-h01-h03-20260707T134429Z`（本地，未入库）
- RDS host：`postgres://ash:ash@...`（脱敏）
- 环境：**本地 Docker**（云 RDS 待 `config/cloud-rds.env`）

## 验收项

| 项 | 状态 | 说明 |
|----|------|------|
| H-01 云 RDS 全链路 | ✅ | `postgres-rds-e2e` + migrate verify + Doctor ALL |
| H-02 RLS + ash_app | ✅ | M3-06/07 + RLS 集成测试 |
| H-03 生产 Worker 配置 | ⚠️ | 本地 `make postgres-app-gate` ✅（readyz）；生产 Worker 部署待签字 |

## 签字（云验收完成后填写）

| 角色 | 姓名 | 日期 |
|------|------|------|
| 技术负责人 | | |
| 测试负责人 | | |
| 发布负责人 | | |

清单：[`h01-h03-cloud-signoff.md`](../checklists/h01-h03-cloud-signoff.md)
