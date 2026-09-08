# Sprint DX44 — Chroma 可选适配器（v3.0 · E3）

> **方案：** Chroma-compatible HTTP（api/v1 heartbeat/collections/upsert/query）；httptest；非默认  
> **Goal:** `ASH_RAG_VECTOR_BACKEND=chroma` + URL 可达时 Available；保留 Qdrant 默认  
> **状态：** ✅ 完成 · **无新表**

## 任务板

| ID | 任务 | 状态 |
|----|------|------|
| DX44-1 | `ChromaClient` heartbeat/create/upsert/query | ✅ |
| DX44-2 | Resolve 接线 + 可选 `ASH_RAG_VECTOR_API_KEY` | ✅ |
| DX44-3 | httptest 契约测 | ✅ |
| DX44-4 | sprint / TODO / CHANGELOG | ✅ |

## Env（增量）

| 变量 | 说明 |
|------|------|
| `ASH_RAG_VECTOR_BACKEND=chroma` | 选用 Chroma 客户端 |
| `ASH_RAG_VECTOR_URL` | 默认 `http://127.0.0.1:8000` |
| `ASH_RAG_VECTOR_API_KEY` | 可选 `X-Chroma-Token` |

## 验收

```bash
go test ./internal/rag/ -count=1 -run 'TestChroma|TestResolveChroma|TestResolveMilvus|TestVectorConfig'
```
