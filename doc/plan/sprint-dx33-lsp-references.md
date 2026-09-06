# Sprint DX33 — LSP References + Hybrid 接线（v2.8）

> **方案：** 有界 `textDocument/references`；Hybrid Query `expandRefs`；LSP 不可用时回退符号表同名行  
> **Goal:** `POST /api/v1/rag/lsp/references` + Query `expandRefs`；Knowledge 展示 `lspAvailable`  
> **状态：** ✅ 完成 · **无新表**

## 任务板

| ID | 任务 | 状态 |
|----|------|------|
| DX33-1 | Session `references` + 上限裁剪 | ✅ |
| DX33-2 | `Service.References` + symbol_table fallback | ✅ |
| DX33-3 | Query `expandRefs` Hybrid 接线 | ✅ |
| DX33-4 | API/OpenAPI + Knowledge `lspAvailable` + 测 | ✅ |

## 约定

| 项 | 值 |
|----|-----|
| API limit | 默认 20，最大 50 |
| expandRefs | 最多 3 个 symbol seed，每个最多 8 条 ref |
| fallback | `source=symbol_table`（同名 `rag_symbols`） |

## 验收

```bash
go test ./internal/rag/ -count=1 -run 'TestHover|TestReferences|TestQueryExpandRefs|TestLSP'
make openapi-check
```
