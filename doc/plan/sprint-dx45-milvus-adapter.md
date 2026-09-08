# Sprint DX45 — Milvus 可选适配器（v3.0 · E3）

> **方案：** Milvus REST v2 兼容（collections/list·create · entities/upsert·search）；httptest；非默认  
> **Goal:** `ASH_RAG_VECTOR_BACKEND=milvus` + URL 可达时 Available；保留 Qdrant 默认  
> **状态：** ✅ 完成 · **无新表**

## 任务板

| ID | 任务 | 状态 |
|----|------|------|
| DX45-1 | `MilvusClient` list/create/upsert/search | ✅ |
| DX45-2 | Resolve 接线 + Bearer API key | ✅ |
| DX45-3 | httptest 契约测 | ✅ |
| DX45-4 | sprint / TODO / CHANGELOG | ✅ |

## Env（增量）

| 变量 | 说明 |
|------|------|
| `ASH_RAG_VECTOR_BACKEND=milvus` | 选用 Milvus 客户端 |
| `ASH_RAG_VECTOR_URL` | 默认 `http://127.0.0.1:19530` |
| `ASH_RAG_VECTOR_API_KEY` | 可选 `Authorization: Bearer …` |

## 验收

```bash
go test ./internal/rag/ -count=1 -run 'TestMilvus|TestResolveMilvus|TestResolveChroma|TestVectorConfig'
```
