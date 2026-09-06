# RAG LSP smoke evidence (DX35)

| Field | Value |
|-------|--------|
| Status | **pass** (exit 0) |
| Platform | MINGW64_NT-10.0-26200 |
| Date | 2026-09-06T18:38:05Z |
| Scope | session / hover / definition / references / expandRefs / harden |
| Live LS | not required (fake-gopls) |

## Env (harden)

| Variable | Default |
|----------|---------|
| `ASH_RAG_LSP_SESSION` | on (`0` = one-shot) |
| `ASH_RAG_LSP_IDLE_SEC` | 30 |
| `ASH_RAG_LSP_TIMEOUT_SEC` | 20 (max 300) |
| `ASH_RAG_LSP_MAX_OPEN_DOCS` | 64 |

## Raw excerpt

```
ok  	github.com/ash-repwiki/ash/internal/rag	15.663s
```
