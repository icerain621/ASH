# KPI 口径对账门禁（§9）

> 对应 [`14-kpi-dashboard-definition.md`](../14-kpi-dashboard-definition.md) §9 指标质量保障。

## 自动化覆盖

| §9 项 | 门禁 | 说明 |
|-------|------|------|
| 埋点字段完整性 | `TestOverviewAggregatesKPIInputs` | KPI-01/04/05/07/08/10 聚合 |
| 样本回放对账 | `TestKPIOverviewMatchesDeriveReplay` | KPI-01 分子与 `ash_run_total{status="finished"}` 一致 |
| derive parity | `TestValidateReplayParity` | 事件 replay 与独立计数对齐 |
| 看板卡片齐全 | `TestKPIOverviewSummaryCatalog` | KPI-01..11 均出现在 overview |
| 场景可重复（R-02） | `TestOverviewScenarioStabilityR02` | KPI-11 + `scenarioStability` |

## 命令

```bash
make kpi-reconcile-gate
make t1-metrics-gate    # 含 API T+1 基线
```

## 仍待人工 / 生产

- SQL 口径评审（BI 建模）
- 与业务侧手工统计偏差 < 5%（上线后 T+1 观察）
