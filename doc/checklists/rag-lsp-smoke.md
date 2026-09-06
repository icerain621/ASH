# RAG LSP 烟测（Sprint DX31–DX35）

```bash
make rag-lsp-smoke
```

写入证据：[`../evidence/rag-lsp-smoke-latest.md`](../evidence/rag-lsp-smoke-latest.md)

| 覆盖 | 说明 |
|------|------|
| Session 复用 / idle / one-shot | DX31 |
| Hover / Definition | DX32 |
| References + `expandRefs` + symbol_table fallback | DX33 |
| Timeout / max-open / missing binary | DX35 harden |
| fake-gopls | 不要求本机 gopls / tsserver |

## Env

| 变量 | 说明 |
|------|------|
| `ASH_RAG_LSP_SESSION` | 默认会话池；`0`/`off` → one-shot |
| `ASH_RAG_LSP_IDLE_SEC` | 空闲回收（默认 30） |
| `ASH_RAG_LSP_TIMEOUT_SEC` | RPC 超时秒（默认 20，最大 300） |
| `ASH_RAG_LSP_MAX_OPEN_DOCS` | 每会话打开文档上限（默认 64） |
| `ASH_RAG_LSP_GOPLS` / `ASH_RAG_LSP_TSSERVER` | 可执行路径（测试用 fake-gopls） |

## 相关

- [`../plan/sprint-dx35-lsp-harden-smoke.md`](../plan/sprint-dx35-lsp-harden-smoke.md)
- [`../plan/v2.8-release-scope.md`](../plan/v2.8-release-scope.md)
