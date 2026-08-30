# Sprint DX2：isolated 强制（方案 C）Implementation Plan

> **方案：** 已批准 **C** = 运行时强制 + DSL `sandbox.minMode` + KPI-19 + 场景对齐  
> **Goal:** hotfix/security 强制 ≥ isolated；danger 低于 isolated 拒绝；KPI-19 覆盖率；无新表。  
> **状态：** ✅ 本批完成

## 任务板

| ID | 任务 | 状态 |
|----|------|------|
| DX2-1 | `ResolveSandboxMode` 地板合并 + Authorize 强化 | ✅ |
| DX2-2 | RouteRequest / Loop / runs 传入 policy + scenarioMin | ✅ |
| DX2-3 | DSL `scenario.sandbox.minMode` + schema/validate | ✅ |
| DX2-4 | KPI-19 metrics + Metrics 页 | ✅ |
| DX2-5 | hotfix/security 场景 + 清单 + docs | ✅ |

## 行为

- `policyProfile ∈ {hotfix, security}` → 有效 mode ≥ `isolated`
- `scenario.sandbox.minMode` 参与地板取 max
- ModeOverride **只能抬高**，不能压过 risk/policy/scenario 地板
- `danger`/`network` 且 mode &lt; isolated → `Authorize` deny
- KPI-19：`harness.tool.routed` 中 risk=danger 且 sandboxMode=isolated / 全部 danger 路由

## 退出标准

- [x] hotfix/security 工具路由 mode=isolated（单测）
- [x] KPI-19 出现在 overview；前端可展示
- [x] 场景 YAML 含 `sandbox.minMode: isolated`
