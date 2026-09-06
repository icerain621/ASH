# 单区域就绪（Sprint DX23 · v2.6）

> **非目标：** Active-Active 多区域、跨区复制、数据驻留产品化。  
> **目标：** 部署可标识区域；备份/恢复与单区域 HA 可操作。

## 区域身份

| 项 | 说明 |
|----|------|
| Env | `ASH_REGION`（空 → `default`） |
| 探针 | `GET /readyz` → `region` |
| Scale | `GET /api/v1/scale/readiness` → `region`（与 readyz 一致） |

```bash
export ASH_REGION=ap-east-1
curl -s "$ASH_WORKER_URL/readyz" | jq .region
```

## 备份 / 恢复（单区域）

复用现有脚本（仓库根目录）：

```bash
make data-backup          # scripts/ash-data-backup.sh（SQLite .ash/ash.db）
make data-backup-verify   # 校验 sha256
make data-backup-smoke
```

Postgres 生产：按 [`postgres-production-config.md`](postgres-production-config.md) 与云厂商快照/PITR；**不要求**跨区域副本。

## 单区域 HA（建议）

| 层 | 建议 | v2.6 Out |
|----|------|----------|
| Worker | 同区域多副本 + 负载均衡；共享 Postgres | 跨区 Worker 联邦 |
| Postgres | 同区域主备 / 托管 HA | 逻辑跨区同步 |
| 对象存储 / 产物 | 同区域 bucket | 跨区复制策略产品化 |
| DNS / 流量 | 单区域入口 | 全局流量调度 |

切换日仍按 [`release-window-runbook.md`](release-window-runbook.md) / MVP BI-5；区域标签写入证据 JSON（`region` 字段）便于事后对账。

## 签字勾选（v2.6）

- [ ] `/readyz.region` 与运维台账区域名一致
- [ ] 备份脚本或云快照演练证据已归档
- [ ] 文档确认：**无**跨区 Active-Active

## 相关

- [`../plan/sprint-dx23-single-region.md`](../plan/sprint-dx23-single-region.md)
- [`../plan/v2.6-release-scope.md`](../plan/v2.6-release-scope.md)
