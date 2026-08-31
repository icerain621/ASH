# Sprint DX9：RAG Hybrid 符号索引（v2.3 · 方案 C）

> **方案：** 已批准 **C** = v2.2 冻结后开 Hybrid：`rag_path_entries` + `rag_symbols` + RebuildSymbols + RRF Query  
> **Goal:** SQL **28** / RLS **48**；`POST /rag/symbols/rebuild`；Hybrid Query 回退 FTS/chunk  
> **状态：** 本批完成 · **新表 2**

## 任务板

| ID | 任务 | 状态 |
|----|------|------|
| DX9-1 | Schema + SQL 28 + RLS 48 + GORM models | ✅ |
| DX9-2 | `RebuildSymbols` 扫盘 upsert + stale 清理 | ✅ |
| DX9-3 | Hybrid Query（path/symbol/text RRF k=60）+ 回退 | ✅ |
| DX9-4 | API + OpenAPI + tenant 越权回归 | ✅ |
| DX9-5 | `make rag-hybrid-smoke` + v2.3 草案 / 水位 | ✅ |

## 行为

- Rebuild：独立于 chunk `Index`；walk repo → upsert path/symbol 行 → 删除磁盘上已不存在的 stale 行
- Query：text + path + symbol 三 lane RRF 融合；空 hybrid 表时回退现有 `fts`/`chunk` 行为
- Out：向量库主路径、ctags 外部依赖

## 退出标准

- [x] `go test ./internal/rag/ -run 'TestRebuildSymbols|TestHybridQuery|TestQueryFallsBack'`
- [x] `make rag-hybrid-smoke`
- [x] `v2.3-release-scope.md` 草案（§1–§6；**未冻结**）
