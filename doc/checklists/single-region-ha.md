# 单区域与多区域探测 / Single-region & multi-region probe

> **DX23 / DX83** · 归属 [`checklists/`](README.md)（若存在）· [`../plan/v4.3-release-scope.md`](../plan/v4.3-release-scope.md)  
> **非目标 / Non-goals：** Active-Active 多区域、跨区复制、数据驻留产品化、全局流量调度。  
> Active-Active, cross-region replication, residency productization, and global traffic steering are out of scope.  
> **目标 / Goals：** 部署可标识区域；探针明确报告非多活；备份/恢复与单区域 HA 可操作。  
> Identify the deployment region, report multi-region as disabled on probes, and keep single-region HA operable.

## 1. 如何读区域 / How to read `region`

| 项 / Item | 说明 / Meaning |
|-----------|----------------|
| Env | `ASH_REGION`（空 / unset → `default`） |
| 探针 / Probe | `GET /readyz` → `region` |
| Scale | `GET /api/v1/scale/readiness` → `region`（须与 readyz 一致 / must match readyz） |
| 控制台 / Console | 规模化页「区域 (ASH_REGION)」行 |

```bash
export ASH_REGION=ap-east-1
# Worker 本机默认 / local worker default:
curl -s http://127.0.0.1:8080/readyz | jq '{region, multiRegion}'
curl -s -H "X-ASH-Space-ID: local" http://127.0.0.1:8080/api/v1/scale/readiness \
  | jq '{region, multiRegion}'
```

**解读 / Reading：** `region` 只是**单区域身份标签**，不表示多活集群或跨区副本。  
`region` is a **single-region identity label**, not an Active-Active cluster or cross-region replica set.

## 2. 多区域关闭 / `multiRegion=disabled`（DX82）

| 项 / Item | 期望 / Expected |
|-----------|-----------------|
| `/readyz.multiRegion` | 恒为 `disabled` / always `disabled` |
| `/api/v1/scale/readiness.multiRegion` | 恒为 `disabled`，与 readyz 一致 / always `disabled`, same as readyz |
| 控制台 / Console | 规模化页「多区域 / Active-Active」显示 `disabled` |

```bash
curl -s http://127.0.0.1:8080/readyz | jq -e '.multiRegion == "disabled"'
```

**禁止误读 / Do not misread：** `multiRegion=disabled` 表示 **本版本不做 Active-Active**，不是「暂时关掉、即将多活」。  
`multiRegion=disabled` means **this release is not Active-Active**, not a temporary off switch.

## 3. 备份 / 恢复（单区域） / Backup & restore (single region)

复用现有脚本（仓库根目录） / Reuse existing scripts from the repo root:

```bash
make data-backup          # scripts/ash-data-backup.sh（SQLite .ash/ash.db）
make data-backup-verify   # 校验 sha256 / verify sha256
make data-backup-smoke
```

Postgres 生产：按 [`postgres-production-config.md`](postgres-production-config.md) 与云厂商快照/PITR；**不要求**跨区域副本。  
Postgres production: follow the production checklist and cloud snapshots/PITR; **cross-region replicas are not required**.

## 4. 单区域 HA（建议） / Single-region HA (suggested)

| 层 / Layer | 建议 / Suggestion | Out |
|------------|-------------------|-----|
| Worker | 同区域多副本 + 负载均衡；共享 Postgres / same-region replicas + LB; shared Postgres | 跨区 Worker 联邦 / cross-region worker federation |
| Postgres | 同区域主备 / 托管 HA / same-region standby or managed HA | 逻辑跨区同步 / logical cross-region sync |
| 对象存储 / 产物 | 同区域 bucket / same-region bucket | 跨区复制产品化 / cross-region replication productization |
| DNS / 流量 | 单区域入口 / single-region ingress | 全局流量调度 / global traffic steering |

切换日仍按 [`release-window-runbook.md`](release-window-runbook.md)；区域标签写入证据 JSON（`region`）便于事后对账。  
On cutover days follow the release-window runbook; put `region` into evidence JSON for reconciliation.

## 5. 签字勾选 / Sign-off checks

### v2.6（DX23）

- [ ] `/readyz.region` 与运维台账区域名一致 / matches the ops ledger region name
- [ ] 备份脚本或云快照演练证据已归档 / backup or snapshot drill evidence archived
- [ ] 文档确认：**无**跨区 Active-Active / docs confirm: **no** cross-region Active-Active

### v4.3（DX83）

- [ ] `/readyz` 与 scale readiness 均含 `multiRegion=disabled` / both probes report `multiRegion=disabled`
- [ ] 运维已知：`region` ≠ 多活；本代不交付 Active-Active / ops know `region` ≠ multi-live; this gen does not ship Active-Active
- [ ] 规模化页可见「多区域 / Active-Active」为 `disabled` / Scale page shows multi-region as `disabled`

## 相关 / Related

- [`../plan/sprint-dx23-single-region.md`](../plan/sprint-dx23-single-region.md)
- [`../plan/sprint-dx82-multi-region-disabled.md`](../plan/sprint-dx82-multi-region-disabled.md)
- [`../plan/sprint-dx83-multi-region-probe-doc.md`](../plan/sprint-dx83-multi-region-probe-doc.md)
- [`../plan/v2.6-release-scope.md`](../plan/v2.6-release-scope.md)
- [`../plan/v4.3-release-scope.md`](../plan/v4.3-release-scope.md)
