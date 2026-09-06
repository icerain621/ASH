# Sprint DX26 — Real Embedder（v2.7）

> 状态：**完成**（2026-09-06）  
> 范围：[`v2.7-release-scope.md`](v2.7-release-scope.md)

## 交付

| # | 项 | 状态 |
|---|-----|------|
| DX26-1 | `OpenAICompatEmbedder`（OpenAI `/v1/embeddings` 兼容） | ✅ |
| DX26-2 | `ResolveEmbedder`：`ASH_EMBED_BASE_URL` 未设 → Hash | ✅ |
| DX26-3 | Profile `embedderKind` / `embedderDim`；Index embed 失败仍 best-effort | ✅ |
| DX26-4 | 包测 + `rag-vector-smoke`；可选 `ASH_EMBED_LIVE=1` live 段 | ✅ |

## 环境变量

| 变量 | 说明 |
|------|------|
| `ASH_EMBED_BASE_URL` | 设则启用 OpenAI-compat（origin / `.../v1` / `.../embeddings`） |
| `ASH_EMBED_API_KEY` | Bearer（可选，视上游） |
| `ASH_EMBED_MODEL` | 默认 `text-embedding-3-small` |
| `ASH_EMBED_DIM` | 配置维数（默认 1536；首次成功响应会学习） |
| `ASH_EMBED_TIMEOUT` | Go duration，默认 `30s` |
| `ASH_EMBED_LIVE` | `1` 时 smoke 跑 live 测（需 BASE_URL） |

## 验收

```bash
go test ./internal/rag/ -count=1 -run 'TestOpenAICompat|TestResolveEmbedder|TestIndexEmbedUsesOpenAI|TestIndexEmbedOpenAIError'
make rag-vector-smoke
make openapi-check
```

## 备注

- **无新表**；不把 Embedding 当发布硬依赖。
- 不在 HTTP 失败时自动混用 Hash（避免与 Qdrant 维数冲突）；未配置 URL 时才用 Hash。
