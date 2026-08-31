# RAG Hybrid 烟测（Sprint DX9）

```bash
make rag-hybrid-smoke
```

| 覆盖 | 说明 |
|------|------|
| `TestRebuildSymbolsUpsertsAndCleansStale` | 扫盘写入 `rag_path_entries` / `rag_symbols` + stale 清理 |
| `TestHybridQueryPrefersSymbolOverNoise` | path/symbol/text RRF 融合，符号优先 |
| `TestQueryFallsBackWhenHybridEmpty` / `TestQueryFallsBackToChunkSearchWhenFTSUnavailable` | 空 hybrid 表或 FTS 不可用时回退 |

## API（可选 live）

```bash
# 需 Worker + repoRoot
curl -s -X POST "$ASH_WORKER_URL/api/v1/rag/symbols/rebuild" \
  -H 'Content-Type: application/json' \
  -d '{"spaceId":"local","repoRoot":"."}'
```

跨 space 拒绝见 `TestCrossSpaceAPIRegression`（`ragSymbolsRebuild`）。

## 相关

- [`../plan/sprint-dx9-rag-hybrid.md`](../plan/sprint-dx9-rag-hybrid.md)
- [`../plan/v2.3-release-scope.md`](../plan/v2.3-release-scope.md)（草案）
- SQL rev **28** · RLS **48**
