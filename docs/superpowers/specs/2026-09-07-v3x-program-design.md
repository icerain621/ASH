# v3.x program design (T1 vector · T2 Quest · T3 IdP/gateway)

## Confirmed (E1–E7 · 2026-09-07)

See `doc/plan/v3.x-program.md` and `doc/plan/v3.0-release-scope.md` §7.

## DX43 — Multi-backend contract

- `VectorBackend` extends `VectorStore` with `Name()`
- `ASH_RAG_VECTOR_BACKEND` default `qdrant`; `mock` available; `milvus` stub until DX45
- `NewService` uses `ResolveVectorStore()`; Profile exposes `vectorBackend`
- Qdrant preserved

## DX44 — Chroma adapter

- `ChromaClient` (api/v1 heartbeat / collections / upsert / query)
- Optional `ASH_RAG_VECTOR_API_KEY` → `X-Chroma-Token`
- httptest contract tests; not default

## DX45 — Milvus adapter

- `MilvusClient` (REST v2 collections/list·create · entities/upsert·search)
- Optional Bearer token; httptest contract tests; not default

## Later

- **DX46** prefer=vector upgrade
- **DX47** smoke + console
- **DX48** freeze + signoff
- **v3.1 / v3.2** Quest / IdP tracks
