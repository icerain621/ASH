# Sprint DX41 — 控制台 remote 水位（v2.9）

> **方案：** Scale readiness 暴露 `ProbeStatus`；Observability / Scale 展示启用与可用性  
> **Goal:** 运维可见 remote 关闭/可用/不可用与 prefer / on_fail；无新表；Doctor 不变  
> **状态：** ✅ 完成 · **无新表**

## 任务板

| ID | 任务 | 状态 |
|----|------|------|
| DX41-1 | Scale `sandboxRemote*` 字段 + ProbeStatus | ✅ |
| DX41-2 | OpenAPI + Scale / Observability UI | ✅ |
| DX41-3 | 前端测 + sprint / TODO / CHANGELOG | ✅ |

## 验收

```bash
go test ./internal/api/ -count=1 -run TestScaleReadiness
cd frontend && npm test -- --run src/pages/ObservabilityPage.test.tsx src/pages/ScalePage.test.tsx
make openapi-check
```
