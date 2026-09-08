# Sprint DX47 — rag-vector smoke + 控制台向量水位（v3.0）

> **方案：** 扩展 `make rag-vector-smoke`（含 Chroma/Milvus/prefer）；Scale / Knowledge / Observability 暴露后端可用性  
> **Goal：** 运维可见 `qdrant|chroma|milvus|mock` 与 prefer 默认；无 live 向量库硬依赖；**无新表**  
> **状态：** ✅ 完成 · **无新表**

## 任务板

| ID | 任务 | 状态 |
|----|------|------|
| DX47-1 | `rag-vector-smoke` 含 DX43–46 过滤 + 证据清单 | ✅ |
| DX47-2 | Scale `ragVector*` + OpenAPI / Profile 契约字段 | ✅ |
| DX47-3 | Scale / Knowledge / Observability UI + 前端测 | ✅ |
| DX47-4 | sprint / TODO / CHANGELOG | ✅ |

## 验收

```bash
make rag-vector-smoke
go test ./internal/api/ -count=1 -run TestScaleReadiness
cd frontend && npm test -- --run src/pages/ObservabilityPage.test.tsx src/pages/ScalePage.test.tsx src/pages/KnowledgePage.test.tsx
make openapi-check
```
