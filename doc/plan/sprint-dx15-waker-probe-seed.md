# Sprint DX15：Waker probe 自动 seed（v2.5 · 方案 C）

> **方案：** 已批准 **C** = Status/Background 自动 seed `doctor_subset` / `kpi_drift`（默认 disabled）+ enable API + 控制台切换  
> **Goal:** 关闭 DX13 方案 A 的手动 Ensure 缺口；探针默认关闭直至显式启用  
> **状态：** 本批完成 · **无新表**

## 任务板

| ID | 任务 | 状态 |
|----|------|------|
| DX15-1 | `SeedProbeDuties` + Status/Background 接线 | ✅ |
| DX15-2 | `POST /waker/duties/{id}/enable` + OpenAPI | ✅ |
| DX15-3 | Observability duty 启用切换 + status 提示 | ✅ |
| DX15-4 | `make waker-smoke` 扩展 + v2.5 范围草案水位 | ✅ |

## 行为

- `Status()` / `StartBackground()` 在 `EnsureStaleRunDuty` 之后调用 `SeedProbeDuties`（不覆盖已有行的 `enabled`）
- 新建 probe duty 默认 `enabled=false`；`ASH_WAKER_ENABLE_PROBES=1` 时新建为 enabled
- `StatusResponse.probesAvailable` / `alertCount` 供控制台展示
- Cancel 闸门与后台永不 cancel 不变

## 退出标准

- [x] `go test ./internal/waker/ ./internal/api/ -count=1 -run 'TestStatusAuto|TestSetDuty|TestProbes|TestWakerDutyEnable'`
- [x] `make waker-smoke`
- [x] `doc/plan/v2.5-release-scope.md` 草案（§1–§4；**未冻结**）
