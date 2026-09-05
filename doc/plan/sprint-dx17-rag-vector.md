# Sprint DX17：RAG Vector POC（v2.5 · 方案 C）

> **方案：** 已批准 **C** = Qdrant HTTP + stub/hash embedder + `rag_vector_refs` + Query 第 4 RRF lane  
> **Goal:** SQL **31** / RLS **51**；向量 lane 可选；无 live Qdrant 硬依赖  
> **状态：** 本批完成 · **列扩展 1**

## 任务板

| ID | 任务 | 状态 |
|----|------|------|
| DX17-1 | SQL 31 `rag_vector_refs` + GORM + RLS | ✅ |
| DX17-2 | Qdrant client + stub embedder + Index/Query vector lane | ✅ |
| DX17-3 | Hybrid Query 第 4 RRF lane（vector） | ✅ |
| DX17-4 | `make rag-vector-smoke` + Profile 向量就绪字段 | ✅ |

## 行为

- `ResolveVectorStore()` / embedder：未配置 Qdrant 时 vector lane 静默跳过
- Index 路径 best-effort embed + upsert `rag_vector_refs`
- Smoke 使用 mock store，无需 live Qdrant

## 退出标准

- [x] `go test ./internal/rag/ -run 'TestQueryVectorLane|TestIndexEmbed|TestQuerySucceedsWithoutVector|TestProfileReportsVector' -count=1`
- [x] `make rag-vector-smoke`
- [x] `v2.5-release-scope.md` 草案（§2 DX17；**未冻结**）
