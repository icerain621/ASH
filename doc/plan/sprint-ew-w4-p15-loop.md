# Sprint EW · P15 Skills/MCP/Hooks 闭环

> 前置：**W0–W3 ✅**  
> 原则：不新造执行面。Run 工具链已走 PreToolUse；本板只补 MCP HTTP 旁路。

| # | Sprint | 状态 |
|---|--------|------|
| **EW41** | MCP execute 走 PreToolUse | ✅ |
| **EW42** | 签字 | ✅ |

## EW41

`POST /mcp/tools/{id}/execute` 读 SpacePolicy `hooks`。`deny` → 409 `MCP_TOOL_HOOK_DENIED`（审批 token 不能绕过）。`ask` 且风险为 low → 走现有一次性 `approvalToken`。配置损坏 fail-closed。无 hooks 时行为不变。

## EW42

CHANGELOG 与本板勾选。无新路径。
