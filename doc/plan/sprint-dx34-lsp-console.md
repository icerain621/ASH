# Sprint DX34 — 控制台 LSP 水位（v2.8）

> **方案：** Observability / Knowledge 样本探针 + Scale 紧凑 `ragLspAvailable`  
> **Goal:** 运维可见 LSP 可用性，并可试跑 hover / definition / references  
> **状态：** ✅ 完成 · **无新表**

## 任务板

| ID | 任务 | 状态 |
|----|------|------|
| DX34-1 | `observability.api` LSP 客户端 | ✅ |
| DX34-2 | 共享 `RagLspProbePanel` | ✅ |
| DX34-3 | Observability / Knowledge 接线 | ✅ |
| DX34-4 | Scale `ragLspAvailable` + OpenAPI + 测 | ✅ |

## 验收

```bash
cd frontend && npm test -- --run src/pages/ObservabilityPage.test.tsx src/pages/KnowledgePage.test.tsx src/pages/ScalePage.test.tsx
make openapi-check
```
