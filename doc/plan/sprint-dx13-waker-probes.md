# Sprint DX13：Waker 多职责探针（v2.4 · 方案 A）

> **方案：** 已批准 **A** = 可选启用 `doctor_subset` / `kpi_drift`（不自动 seed）  
> **Goal:** 探针写 `waker_duty_runs`；失败/超阈 → queue 发现项；永不 cancel  
> **状态：** 本批完成 · **无新表**

## 任务板

| ID | 任务 | 状态 |
|----|------|------|
| DX13-1 | probes.go：DoctorRunner + KPI-17 + 配置解析 | ✅ |
| DX13-2 | RunDueDuties/RunDuty + Ensure* + Queue 合并 | ✅ |
| DX13-3 | NewHandler DoctorRunner 适配 + `make waker-smoke` | ✅ |
| DX13-4 | sprint / TODO / CHANGELOG / v2.4 草案水位 | ✅ |

## 行为

- `doctor_subset`：默认 suite `M4`；可选 `caseIds`；无 runner → `skipped`
- `kpi_drift`：默认 `KPI-17` threshold `50`；flag only
- Status / Worker **不** seed 这两种 duty
- UI / `v2.4-signoff` 属 **DX14**

## 退出标准

- [x] `go test ./internal/waker/ -count=1`
- [x] `make waker-smoke`
