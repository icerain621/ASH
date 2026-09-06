# Sprint DX35 — LSP harden + smoke（v2.8）

> **方案：** 可配超时 / 打开文档上限 / 缺二进制快速失败；`make rag-lsp-smoke` + 证据  
> **Goal:** 无 live LS 依赖即可门禁；为 DX36 `v2.8-signoff` 铺路  
> **状态：** ✅ 完成 · **无新表**

## 任务板

| ID | 任务 | 状态 |
|----|------|------|
| DX35-1 | `ASH_RAG_LSP_TIMEOUT_SEC` / `MAX_OPEN_DOCS` | ✅ |
| DX35-2 | 缺二进制 → `ErrLSPUnavailable`（Stat） | ✅ |
| DX35-3 | `scripts/rag-lsp-smoke.sh` + Makefile | ✅ |
| DX35-4 | checklist / evidence / TODO / CHANGELOG | ✅ |

## 验收

```bash
make rag-lsp-smoke
# → doc/evidence/rag-lsp-smoke-latest.md
```
