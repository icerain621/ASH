# Sprint DX76 — 进程内 Indexer / In-process indexer

> **前置 / Prerequisite：** DX75  
> **状态 / Status：** ✅  
> **做法 / Approach：** `ASH_RAG_INDEXER_GRPC_ADDR` 为空则不监听；设置后本机 gRPC 委托现有 `rag.Service`。无新表、不拆进程。  
> Empty `ASH_RAG_INDEXER_GRPC_ADDR` means no listen. When set, loopback gRPC delegates to the existing `rag.Service`. No new tables and no process split.

## 任务板 / Board

| ID | 任务 / Task | 状态 |
|----|-------------|------|
| DX76-1 | `StartIndexerServer` + IndexRepo/Query 适配 | ✅ |
| DX76-2 | config + worker 接线（默认空） | ✅ |
| DX76-3 | 单测：空地址 noop / Index+Query / 空 text | ✅ |

## 验收 / Verify

```bash
go test ./internal/rag/ ./internal/config/ -count=1 -run 'Indexer|RagIndexer'
```
