# 发布窗口运行手册（MVP §8）

> 模板：发布前填写值班表并归档至 `doc/evidence/release-window-YYYY-MM-DD.md`。

## 1. 发布窗口

| 项 | 值 |
|----|-----|
| 窗口开始（UTC+8） | 2026-07-10 02:00 |
| 窗口结束 | ____-__-__ __:__ |
| 维护类型 | Postgres 切换 / 应用滚动 / 配置变更 |

## 2. 值班 roster

| 角色 | 姓名 | 联系方式 | 备岗 |
|------|------|----------|------|
| 发布负责人 | 发布负责人（占位） | | |
| 技术 on-call | 技术 on-call（占位） | | |
| DBA / 平台 | DBA（占位） | | |
| 测试 | 测试负责人（占位） | | |

## 3. 发布前 24h 门禁

```bash
make release-window-gate        # §8 本地聚合（含 mvp-signoff / backup / T+0/T+1）
make mvp-signoff
make production-config-gate
make pre-migrate-gate          # 有 ash.db 时备份 + migrate plan
bash scripts/source-cloud-rds-env.sh   # 云切换
make cloud-acceptance          # H-01~H-02
```

| 门禁 | 负责人 | 结果 | 证据路径 |
|------|--------|------|----------|
| mvp-signoff | | ☐ | `doc/evidence/mvp-signoff-latest.md` |
| release-window-gate | | ☐ | `doc/evidence/release-window-latest.md` |
| cloud-acceptance | | ☐ | `doc/evidence/cloud-h01-h03-latest.md` |
| rollback-drill | | ☐ | `doc/evidence/rollback-drill-latest.md` |

## 4. 切换当日步骤（Postgres）

1. `make data-backup`
2. 停 Worker（SQLite）
3. 最终 `migrate copy` + `verify`
4. 设置 `ASH_DATABASE_APP_URL` + RLS env
5. 启 Worker → `make worker-local-gate` 或云 `make live-smoke`
6. `make t0-alert-gate`

## 5. 回滚触发与动作

见 [`postgres-rds-e2e.md`](checklists/postgres-rds-e2e.md) §8。

## 6. T+1 观察（24h）

```bash
make t1-metrics-gate
# 生产环境补充：KPI 看板、反馈队列、告警 evaluate
```

| 指标 | 阈值 | 实际 | 结论 |
|------|------|------|------|
| Run 成功率 | ≥ 95% | | |
| CI 诊断采纳率 | ≥ 目标 | | |
| P95 API 延迟 | ≤ SLO | | |
| 活跃告警数 | 0 critical | | |

## 7. 签字

| 角色 | 姓名 | 日期 |
|------|------|------|
| 发布负责人 | 发布负责人（占位） | 2026-07-07 |
| 值班 SRE | 技术 on-call（占位） | 2026-07-07 |
