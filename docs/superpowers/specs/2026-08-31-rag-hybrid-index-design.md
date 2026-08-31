# Design: RAG Hybrid Path + Symbol Index (Sprint DX9)

> Status: approved for planning (2026-08-31)  
> Version lane: **v2.3 draft** (v2.2 frozen; Hybrid was Out of v2.2)  
> Related: `internal/rag`, `POST /api/v1/rag/*`, agentic roadmap「索引增强」

## Goal

Add a **thin Hybrid retrieval layer** on top of existing chunk FTS: merge **path**, **symbol**, and **text** hits in one Query. No vector DB. Symbol/path tables are **decoupled** from chunk `Index` and filled via an explicit rebuild API.

## Non-goals

- Vector / embedding stores (Chroma, Qdrant, etc.)
- ctags / tree-sitter / language-server extraction
- Background cron rebuild (API only this sprint)
- Changing chunk `Index` to auto-write symbol/path tables
- Automatic git tags

## Architecture

```
                    ┌─────────────────────┐
  rebuild API ─────►│ rag_path_entries    │──┐
  (scan repo)       │ rag_symbols         │  │
                    └─────────────────────┘  │
                                             ▼
  query API  ──► FTS/chunk ──┐         merge (RRF / weights)
                 path lookup─┼──────────► Hit[]  retrievalMode=hybrid
                 symbol lookup┘
```

- Package: extend `internal/rag` (optional `rebuild.go` / small helper file; no new top-level module required).
- Chunk FTS (`rag_chunks` / FTS5 / Postgres `search_vector`) remains the text path.
- When path/symbol tables are empty or unavailable, Query **silently falls back** to `fts` or `chunk` (no 5xx).

## Schema (SQL rev 27 → 28, RLS 46 → 48)

### `rag_path_entries`

| Column | Notes |
|--------|--------|
| id | PK |
| space_id | tenant; RLS |
| repo_root | absolute/normalized root |
| path | relative path |
| basename | file name for basename match |
| digest | content or path digest for stale cleanup |
| updated_at | |

Unique: `(space_id, repo_root, path)`.

### `rag_symbols`

| Column | Notes |
|--------|--------|
| id | PK |
| space_id | tenant; RLS |
| repo_root | |
| path | relative |
| name | symbol identifier |
| kind | `func` \| `class` \| `type` \| `var` \| `unknown` |
| line | 1-based |
| digest | |
| updated_at | |

Indexes: `(space_id, repo_root, name)`, `(space_id, repo_root, path)`.

Both tables: Postgres migration + `000013` RLS list + GORM models + SQLite AutoMigrate parity. Doctor schema/RLS probes updated if they assert rev/policy counts.

### Extraction

Reuse existing `symbolPatterns` / `lineSymbol` (Go / TS / JS). No new parser dependency this sprint.

## API

| Method | Path | Behavior |
|--------|------|----------|
| POST | `/api/v1/rag/symbols/rebuild` | Body `{spaceId, repoRoot}`: walk repo, upsert both tables, delete stale rows for that space+root. Permission: same as `rag/index`. |
| POST | `/api/v1/rag/query` | If hybrid data present: three-way recall + merge; `retrievalMode: "hybrid"`. Optional `prefer: path\|symbol\|text` (default balanced). |
| GET | `/api/v1/rag/profile` | Add `hybridAvailable`, path/symbol counts (or equivalent). |

Error codes:

- Rebuild I/O / persist failure → `RAG_SYMBOLS_REBUILD_FAILED`
- Invalid `prefer` / bad body → `INVALID_REQUEST` / `400`
- Cross-space → existing tenant rejection
- Empty hybrid tables → fallback modes, not an error

OpenAPI + swag annotations required; `make openapi-check`.

## Merge algorithm

1. Run text path (existing FTS or LIKE) → top N.
2. Path table: basename equality / path substring or prefix → top N.
3. Symbol table: name equality / prefix → top N.
4. Fuse with **RRF (k=60)** as the sole merge algorithm; apply `prefer` as a multiplicative boost on that lane’s RRF contribution.
5. Dedupe by citation `ref` (path#symbol:lines or path:lines); keep highest score.
6. Return unified `Hit` shape (existing fields).

## Testing

- Package: rebuild upsert + stale cleanup; RRF dedupe; empty-table fallback; `prefer=path|symbol` bias.
- API: cross-space rebuild/query denied (align existing rag tenant tests).
- Smoke checklist: `doc/checklists/rag-hybrid-smoke.md`; optional `make rag-hybrid-smoke`.
- Out of scope: large-repo P95 load test.

## Docs / release

- Sprint plan: `doc/plan/sprint-dx9-rag-hybrid.md`
- Draft: `doc/plan/v2.3-release-scope.md` (DX9 in scope)
- CHANGELOG `[Unreleased]`; TODO/PLAN watermarks

## Acceptance checklist

- [ ] SQL 28 + RLS policies for both tables
- [ ] rebuild + hybrid query + profile fields
- [ ] Tests green; openapi-check green
- [ ] Smoke checklist present
- [ ] No vector / ctags / auto-index coupling
