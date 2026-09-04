# Sprint DX16：ctags 符号索引（v2.5 · 方案 C）

> **方案：** 已批准 **C** = v2.4 冻结后开 ctags：`SymbolIndexer` 可插拔 + `rag_symbols.source` + regex 回退  
> **Goal:** SQL **30**（无 RLS bump）；`RebuildSymbolsResponse.symbolSource`；`ASH_RAG_CTAGS` / `CTAGS`  
> **状态：** 本批完成 · **列扩展 1**

## 任务板

| ID | 任务 | 状态 |
|----|------|------|
| DX16-1 | SQL 30 `rag_symbols.source` + GORM | ✅ |
| DX16-2 | `SymbolIndexer`（ctags + regex fallback）+ Rebuild 接线 | ✅ |
| DX16-3 | API + OpenAPI `symbolSource` + `make rag-hybrid-smoke` | ✅ |

## 行为

- `ResolveSymbolIndexer()`：`ASH_RAG_CTAGS=1` 时用 ctags 子进程；单文件失败回退 regex
- 响应 `symbolSource`：`ctags` 或 `regex`（任一文件回退则为 `regex`）
- 每符号行 `rag_symbols.source` 记录实际 indexer

## 退出标准

- [x] `go test ./internal/rag/ -run 'TestRebuildSymbolsUpserts|TestRebuildSymbolsFallsBack|TestHybridQuery|TestQueryFallsBack'`
- [x] `make rag-hybrid-smoke`
- [x] `make swagger && make openapi-check`
- [x] `v2.5-release-scope.md` 草案（§2 DX16；**未冻结**）
