# Sandbox isolated 强制（DX2 / KPI-19）

> danger 工具与 hotfix/security 场景须达到 `sandboxMode=isolated`。  
> 度量：Metrics overview **KPI-19**（`harness.tool.routed` 中 risk=danger 且 mode=isolated 占比）。

## 步骤

| # | 动作 | 期望 |
|---|------|------|
| 1 | `go test ./internal/sandbox/... ./internal/runs/ -run 'Sandbox\|Hotfix\|ForceIsolated' -count=1` | 绿 |
| 2 | `go test ./internal/metrics/ -run KPI19 -count=1` | KPI-19 聚合绿 |
| 3 | 跑 hotfix/security 场景后看 Metrics | KPI-19 → 100%（无 workspace-write danger） |
| 4 | `ASH_SKIP_SANDBOX=1 make sandbox-smoke` | 仍可通过（process jail） |

## 规则摘要

- `policyProfile ∈ {hotfix, security}` → 有效 mode ≥ isolated
- `scenario.sandbox.minMode` 参与地板
- ModeOverride 只能抬高，不能压过 risk/policy/scenario 地板
- danger/network 且 mode &lt; isolated → `policy.denied`
