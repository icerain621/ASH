# Sprint DX10：Hybrid 接线（Run / CLI / Knowledge）

> **方案：** 已批准 **C** = Run/Quest 准备阶段 rebuild + `ash rag rebuild` + Knowledge Hybrid 面板  
> **Goal:** Hybrid 索引在执行路径自动补齐；控制台可查看/手动重建；**暂不强制 git commit**  
> **状态：** 本批实现中 · **无新表**

## 任务板

| ID | 任务 | 状态 |
|----|------|------|
| DX10-1 | `prepareExecutionContext` 在 Index 后 best-effort `RebuildSymbols` + 事件 | ✅ |
| DX10-2 | `ash rag rebuild` CLI | ✅ |
| DX10-3 | Knowledge RAG Hybrid 面板 + rebuild 按钮 | ✅ |
| DX10-4 | Scale readiness Hybrid 字段 | ✅ |

## 行为

- Run 准备：`rag.indexed` → `rag.symbols_rebuilt`（失败则 warn）→ `rag.retrieved`（有表则 `retrievalMode=hybrid`）
- CLI：`ash rag rebuild [--repo .] [--space local]` → JSON `{paths,symbols,files}`
- Knowledge：展示 profile Hybrid 水位；按钮调用 `POST /rag/symbols/rebuild`
