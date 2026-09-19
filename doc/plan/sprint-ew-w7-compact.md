# Sprint EW · P06 Compact 语义

> 前置：EW23 已投影 `harness.compaction`；Run 侧 spill 已有  
> 原则：会话 slash 复用同一事件类型，不新造 API。

| # | Sprint | 状态 |
|---|--------|------|
| **EW71** | `/compact` 压缩 transcript + 发 compaction 事件 | ✅ |
| **EW72** | Details 回放摘要 + 签字 | ✅ |

## EW71

Builtin `/compact`：丢掉先前 turns/replies，保留本命令；有 `runId` 时 Append `harness.compaction`（summary 含 prior 计数）。

## EW72

Details 对 `kind=compact` 突出 summary；CHANGELOG + 板勾选。
