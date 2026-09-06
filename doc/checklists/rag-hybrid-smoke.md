# RAG Hybrid 烟测（Sprint DX9 / DX16 / DX20）

```bash
make rag-hybrid-smoke
```

| 覆盖 | 说明 |
|------|------|
| `TestRebuildSymbolsUpsertsAndCleansStale` | 扫盘写入 + stale 清理（强制 regex） |
| `TestRebuildSymbolsFallsBackWhenCtagsFails` | ctags 失败 → regex |
| `TestRebuildSymbolsTreesitterSource` | 默认/强制 treesitter；混合仓保留 preferred |
| `TestTreeSitterIndexer*` / `TestResolveSymbolIndexer*` | Go AST / JS·TS / YAML + 解析器选择 |
| `TestHybridQueryPrefersSymbolOverNoise` | path/symbol/text RRF 融合，符号优先 |
| `TestQueryFallsBackWhenHybridEmpty` / `TestQueryFallsBackToChunkSearchWhenFTSUnavailable` | 空 hybrid 表或 FTS 不可用时回退 |

## Env

| 变量 | 作用 |
|------|------|
| `ASH_RAG_SYMBOL_INDEXER` | `treesitter` \| `ctags` \| `regex`（最高优先级） |
| `ASH_RAG_CTAGS` | `0`→regex；`1`→ctags；unset→treesitter（无 SYMBOL_INDEXER 时） |
| `CTAGS` | ctags 可执行路径（含 fixture 脚本） |

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
- [`../plan/sprint-dx16-rag-ctags.md`](../plan/sprint-dx16-rag-ctags.md)
- [`../plan/sprint-dx20-rag-treesitter.md`](../plan/sprint-dx20-rag-treesitter.md)
- [`../plan/v2.6-release-scope.md`](../plan/v2.6-release-scope.md)
- SQL rev **31** · RLS **51**（DX20 无 bump）
