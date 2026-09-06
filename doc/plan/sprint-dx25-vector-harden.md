# Sprint DX25 — Vector harden（v2.7）

> 状态：**完成**（2026-09-06）  
> 范围：[`v2.7-release-scope.md`](v2.7-release-scope.md) · 设计 [`docs/superpowers/specs/2026-09-06-v27-dx25-dx30-design.md`](../../docs/superpowers/specs/2026-09-06-v27-dx25-dx30-design.md)

## 交付

| # | 项 | 状态 |
|---|-----|------|
| DX25-1 | `prefer=vector` → 有 vector hits 时 `retrievalMode=vector`；无命中回退 hybrid/text | ✅ |
| DX25-2 | Qdrant Upsert/Search 前 `ensureCollection`（GET 缺失则 PUT Cosine） | ✅ |
| DX25-3 | Profile：hybrid+vector / vector 诚实默认模式 | ✅ |
| DX25-4 | 包测 + `make rag-vector-smoke`；OpenAPI prefer 含 `vector` | ✅ |

## 验收

```bash
go test ./internal/rag/ -count=1 -run 'TestQueryVectorLane|TestQueryPreferVector|TestQdrantClientCreatesCollection|TestProfileReports'
make rag-vector-smoke
make openapi-check
```

## 备注

- **无新表** / 无 RLS bump。
- 完整 LSP 仍 Out；下一 Sprint **DX26** Embedder。
