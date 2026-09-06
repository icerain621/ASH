# Sprint DX31 — LSP 工作区短生命周期会话（v2.8 · C1）

> **方案：** 复用 stdio LSP 进程（按 workspace + server）；空闲超时回收；非 IDE 守护  
> **Goal:** `documentSymbol` 在同仓多文件重建时只 `initialize` 一次；`ASH_RAG_LSP_SESSION=0` 可回退 one-shot  
> **状态：** ✅ 完成 · **无新表**

## 任务板

| ID | 任务 | 状态 |
|----|------|------|
| DX31-1 | `lspSessionPool`：按 root+server 复用；串行 RPC；idle 回收 | ✅ |
| DX31-2 | `LSPIndexer` 接线 + Rebuild 注入 workspaceRoot + Close | ✅ |
| DX31-3 | fake-gopls 事件日志；复用 / idle / one-shot 测 | ✅ |
| DX31-4 | sprint / TODO / CHANGELOG / 范围决议 | ✅ |

## 环境变量

| 变量 | 说明 |
|------|------|
| `ASH_RAG_LSP_SESSION` | 默认开启会话池；`0`/`false`/`off` → one-shot（DX29 行为） |
| `ASH_RAG_LSP_IDLE_SEC` | 空闲回收秒数（默认 **30**；最小 1） |
| `ASH_RAG_LSP_GOPLS` / `ASH_RAG_LSP_TSSERVER` | 同 DX29 |

## 验收

```bash
go test ./internal/rag/ -count=1 -run 'TestLSP|TestRebuildSymbolsLSP|TestResolveSymbolIndexer|TestLSPSession'
```

## 退出标准

- [x] 同 workspace 两次 `IndexFile` 仅一次 `initialize`（fake-gopls 事件日志）
- [x] idle 超时后新会话再次 `initialize`
- [x] `ASH_RAG_LSP_SESSION=0` 保持 one-shot
- [x] 既有 LSP/rebuild 测仍绿；无新表
