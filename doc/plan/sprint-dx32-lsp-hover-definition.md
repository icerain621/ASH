# Sprint DX32 — LSP Hover / Definition（v2.8 · C1/C3）

> **方案：** 复用 DX31 会话池；`textDocument/hover` + `definition`；RAG 内部 API（非公开 `/lsp/*`）  
> **Goal:** `POST /api/v1/rag/lsp/hover|definition` + Profile `lspAvailable`；fake-gopls 覆盖  
> **状态：** ✅ 完成 · **无新表**

## 任务板

| ID | 任务 | 状态 |
|----|------|------|
| DX32-1 | Session `hover` / `definition` + 解析 | ✅ |
| DX32-2 | `Service.Hover` / `Definition` | ✅ |
| DX32-3 | API + OpenAPI + Profile `lspAvailable` | ✅ |
| DX32-4 | fake-gopls + 单测 + docs | ✅ |

## API

| 方法 | 路径 | 权限 |
|------|------|------|
| POST | `/api/v1/rag/lsp/hover` | `rag.query` |
| POST | `/api/v1/rag/lsp/definition` | `rag.query` |

Body：`repoRoot` + `path` + `line`(1-based) + optional `character`(0-based) / `text` / `spaceId`

## 验收

```bash
go test ./internal/rag/ -count=1 -run 'TestLSP|TestHover|TestParseHover|TestRebuildSymbolsLSP|TestLSPSession'
make openapi-check
```
