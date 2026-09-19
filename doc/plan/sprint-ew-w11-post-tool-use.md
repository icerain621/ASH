# Sprint EW · PostToolUse

> 前置：PreToolUse MVP（EW11–EW13）  
> 原则：工具已执行后的声明式钩子；`deny` 中止后续步骤，不回滚副作用。

| # | Sprint | 状态 |
|---|--------|------|
| **EW111** | `PostToolUse` 事件 + Run 接入 | ✅ |
| **EW112** | FE/附录 + 签字 | ✅ |

## EW111

`ash.hooks.v1` 增加 `PostToolUse`。工具成功返回后 Evaluate；非默认 allow 发 `hook.post_tool_use` + `hook.decision`。`deny` → 步骤失败并 fail run。`ask` 仅审计（不打断，副作用已发生）。

## EW112

Trajectory 识别；附录更新；CHANGELOG。
