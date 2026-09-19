# Sprint EW · Agent → MemoryLink 深链

> 前置：MemoryLinkPanel 已在 Quest/Reviews；原型 `agent-goto-session-memory`  
> 原则：只加跳转与记忆页「关联」页签，无新 API。

| # | Sprint | 状态 |
|---|--------|------|
| **EW61** | Agent 头「记忆」→ `/memory?tab=links&runId=` | ✅ |
| **EW62** | 签字 | ✅ |

## EW61

Chat 头按钮带当前 `runId` 进记忆页「关联」页签，预填并渲染 `MemoryLinkPanel`。无 runId 仍可进页签后手填。

## EW62

CHANGELOG + 板勾选。
