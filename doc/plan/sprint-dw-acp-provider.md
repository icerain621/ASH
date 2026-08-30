# Sprint DW：ACP Provider 骨架（v2.1 · 方案 C）

> **方案：** 已批准 **C** = ACP 真接线骨架 + `/providers/agent` 探测 + 不可用回退  
> **Goal:** `provider.kind=acp_sdk` 不再静默等同 static；Probe/Execute 走 `ASH_ACP_ENDPOINT`；失败发 `provider.fallback`  
> **状态：** 本批完成 · **无新表**

## 任务板

| ID | 任务 | 状态 |
|----|------|------|
| DW-1 | `agentexec.ACPExecutor` + `ProbeACP` + `Resolve(acp_sdk)` | ✅ |
| DW-2 | runs SelectProvider 按 ACP probe 回退；API `acp` 字段 | ✅ |
| DW-3 | OpenAPI / liveGate `ASH_ACP_E2E` / 清单 | ✅ |
| DW-4 | v2.1 scope 草案 + TODO/PLAN | ✅ |

## 行为

- `Resolve("acp_sdk")` → `ACPExecutor`（Adapter `acp_sdk`）
- 未配置 / 端点不可达 → static + `Fallback` + probe message（与 ExecGo 对称）
- 端点可达 → 用 ACP；`POST {ASH_ACP_ENDPOINT}/v1/tasks`（schema `ash.acp.task.v1`）
- `GET /api/v1/providers/agent` 增加 `acp` / `acpE2EEnabled`

## 环境变量

| 变量 | 说明 |
|------|------|
| `ASH_ACP_ENDPOINT` / `ASH_ACP_URL` | ACP 控制面 HTTP 基址（必需） |
| `ASH_ACP_BIN` | 可选；仅探测提示 |
| `ASH_ACP_AGENT_ID` | 默认 `ash-acp` |
| `ASH_ACP_E2E=1` | liveGateHints 标记 |

## 退出标准

- [x] 未配置时 harness `acp_sdk` → fallback static（可测）
- [x] httptest 可达时 Probe OK + Execute 成功（可测）
- [x] OpenAPI 含 `acp`；清单 `acp-provider.md`
