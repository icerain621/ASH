# Sprint DX46 — Hybrid / prefer=vector 升格（v3.0 · E4）

> **方案：** `prefer=vector` 可选强路径；显式回退原因；`hybrid+vector`；可选 env 默认 prefer；**不强制**云向量（E4）  
> **Goal：** 有命中则 `retrievalMode=vector`；否则回退 hybrid/text 并返回 `vectorFallback`  
> **状态：** ✅ 完成 · **无新表**

## 任务板

| ID | 任务 | 状态 |
|----|------|------|
| DX46-1 | `effectivePrefer` + `ASH_RAG_VECTOR_DEFAULT_PREFER` | ✅ |
| DX46-2 | `prefer=vector` 回退硬化（`store_unavailable` / `no_refs` / `no_hits`） | ✅ |
| DX46-3 | 接受 `prefer=hybrid+vector`（RRF 抬升 vector lane） | ✅ |
| DX46-4 | 响应元数据 `vectorAvailable` / `vectorFallback` / `preferApplied` | ✅ |
| DX46-5 | 单测 + OpenAPI + sprint / TODO / CHANGELOG | ✅ |

## Env（增量）

| 变量 | 说明 |
|------|------|
| `ASH_RAG_VECTOR_DEFAULT_PREFER` | 请求未带 `prefer` 时的可选默认（如 `vector`）；空则保持历史 hybrid/FTS |

## 验收

```bash
go test ./internal/rag/ -count=1 -run 'TestQueryPrefer|TestEffectivePrefer|TestHybrid|TestVector'
make openapi-check
```
