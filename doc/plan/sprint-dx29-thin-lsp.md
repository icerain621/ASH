# Sprint DX29 — 薄 LSP 符号索引（v2.7 · B8 必做）

> 状态：**完成**（2026-09-06）  
> 范围：[`v2.7-release-scope.md`](v2.7-release-scope.md)

## 交付

| # | 项 | 状态 |
|---|-----|------|
| DX29-1 | `LSPIndexer` one-shot `documentSymbol`（gopls / typescript-language-server） | ✅ |
| DX29-2 | `source=lsp`；`ASH_RAG_SYMBOL_INDEXER=lsp` / `ASH_RAG_LSP=1` | ✅ |
| DX29-3 | 单文件回退 lsp→treesitter→regex；fake-gopls 夹具 | ✅ |
| DX29-4 | OpenAPI enum + `rag-hybrid-smoke` + docs | ✅ |

## 约定

| 变量 | 说明 |
|------|------|
| `ASH_RAG_SYMBOL_INDEXER=lsp` | 强制 LSP |
| `ASH_RAG_LSP=1` | 未强制时：PATH 上有 gopls/tsserver 则优先 lsp，否则 treesitter |
| `ASH_RAG_LSP_GOPLS` | gopls 可执行路径（测试用 fake-gopls） |
| `ASH_RAG_LSP_TSSERVER` | typescript-language-server 路径 |
| 默认 | 仍为 **treesitter**（DX20）；完整 hover/defs Out → v2.8 |

## 验收

```bash
go test ./internal/rag/ -count=1 -run 'TestLSP|TestRebuildSymbolsLSP|TestResolveSymbolIndexer'
make rag-hybrid-smoke
make openapi-check
```
