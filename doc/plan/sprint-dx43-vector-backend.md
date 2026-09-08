# Sprint DX43 — 向量多后端契约（v3.0 · E3/E7）

> **方案：** `VectorBackend` + `ASH_RAG_VECTOR_BACKEND`；默认 qdrant；mock 可用；chroma/milvus stub  
> **Goal:** `ResolveVectorStore`；Profile `vectorBackend`；无新表；Qdrant 行为不变  
> **状态：** ✅ 完成 · **无新表**

## 任务板

| ID | 任务 | 状态 |
|----|------|------|
| DX43-1 | `VectorBackend` / Config / ProbeVectorStatus | ✅ |
| DX43-2 | mock 内存后端；chroma/milvus stub | ✅ |
| DX43-3 | `NewService` → `ResolveVectorStore`；Profile 字段 | ✅ |
| DX43-4 | 单测 + sprint / TODO / CHANGELOG | ✅ |

## Env

| 变量 | 说明 |
|------|------|
| `ASH_RAG_VECTOR_BACKEND` | 默认 `qdrant`；`mock` \| `chroma` \| `milvus` |
| `ASH_RAG_VECTOR_URL` | 可选 base URL（qdrant 仍可读 `ASH_QDRANT_URL`） |

## 验收

```bash
go test ./internal/rag/ -count=1 -run 'TestVectorConfig|TestResolveVector|TestProbeVector|TestProfileReportsVectorBackend'
```
