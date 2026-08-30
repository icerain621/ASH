# 演进 KPI + 移动审阅（Sprint DU）

> 验收：Metrics 展示 KPI-17~19；手机可打开 `/ui/m/reviews` 审批准/拒绝。

## 步骤

| # | 动作 | 期望 |
|---|------|------|
| 1 | `go test ./internal/metrics/ -run 'KPI17\|KPI18\|Catalog' -count=1` | KPI-17/18 聚合绿；catalog 含 17/18/19 |
| 2 | 打开 `/ui/metrics` | 「演进 KPI（17–19）」区块含积压 / 回滚率 / danger 覆盖率 |
| 3 | 打开 `/ui/m/reviews`（或评审页「移动审阅」链接） | 精简壳无全站 Tab；列表可批准/拒绝 |
| 4 | 造 `in_review` harness/patch 后刷新 Metrics | KPI-17 上升；&gt;7 天项计入描述 |

## 口径摘要

- **KPI-17**：`in_review` harness + scenario_patch 数量；目标 &lt; 50
- **KPI-18**：`rolled_back / (promoted + rolled_back)`；目标 &lt; 2%
- **KPI-19**：danger → isolated 覆盖率（DX2）

## 相关

- `internal/metrics/service.go` · `frontend/src/pages/MobileReviewsPage.tsx`
- 计划：`doc/plan/sprint-du-evolve-kpi.md`
