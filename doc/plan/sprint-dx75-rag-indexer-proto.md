# Sprint DX75 — RAG Indexer proto / Indexer contract

> **前置 / Prerequisite：** DX74  
> **状态 / Status：** ✅  
> **做法 / Approach：** 新增 `proto/ash/v1/rag_indexer.proto`（`IndexRepo` / `Query`），字段对齐现有 HTTP RAG；`make proto-check` 绿。本 Sprint **不**开监听（→ DX76）。  
> Add `rag_indexer.proto` (`IndexRepo` / `Query`) aligned with the HTTP RAG shapes. `make proto-check` must pass. No listen in this sprint (→ DX76).

## 任务板 / Board

| ID | 任务 / Task | 状态 |
|----|-------------|------|
| DX75-1 | `rag_indexer.proto` + Go 生成物 | ✅ |
| DX75-2 | `make proto-lint` / `make proto-check` | ✅ |
| DX75-3 | 板 / TODO / CHANGELOG | ✅ |

## 验收 / Verify

```bash
make proto-check
```
