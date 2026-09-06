# Sprint DX20：tree-sitter 符号索引（v2.6）

> **方案：** 已批准 **B1** = `TreeSitterIndexer` 纯 Go 解析（go/ast + TS/JS/YAML）；无 CGO；LSP Out  
> **Goal：** 默认 `symbolSource=treesitter`；`ASH_RAG_SYMBOL_INDEXER` 可强制；保留 ctags/regex；**无新表**  
> **状态：** ✅ 完成（代码）

## 任务板

| ID | 任务 | 状态 |
|----|------|------|
| DX20-1 | `TreeSitterIndexer`（Go AST / JS·TS / YAML） | ✅ |
| DX20-2 | `ResolveSymbolIndexer` 默认 treesitter；env 覆盖 | ✅ |
| DX20-3 | Rebuild `symbolSource` 聚合（任一 preferred 成功则保留） | ✅ |
| DX20-4 | OpenAPI enum + `rag-hybrid-smoke` + 包测 | ✅ |
| DX20-5 | sprint / CHANGELOG / TODO | ✅ |

## 约定

- 默认（`ASH_RAG_CTAGS` unset）：`treesitter`
- `ASH_RAG_CTAGS=0`：仍强制 `regex`（DX16 兼容）
- `ASH_RAG_CTAGS=1`：强制 ctags（缺失则 regex）
- `ASH_RAG_SYMBOL_INDEXER=treesitter|ctags|regex`：最高优先级
- 不支持的扩展（如 `.md`）→ 单文件回退 regex；响应在至少一文件 preferred 成功时仍报 `treesitter`
- 实现为 **纯 Go**（非 CGO tree-sitter 绑定），占位名 `treesitter`

## 退出标准

- [x] Go/TS/YAML 包测绿
- [x] `make rag-hybrid-smoke` 覆盖 treesitter
- [x] OpenAPI `symbolSource` 含 `treesitter`
- [x] 无 SQL/RLS bump
