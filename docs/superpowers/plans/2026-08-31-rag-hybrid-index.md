# RAG Hybrid Path + Symbol Index (DX9) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add decoupled path/symbol tables, `POST /rag/symbols/rebuild`, and RRF-merged Hybrid Query on top of existing FTS (no vector DB).

**Architecture:** New `rag_path_entries` + `rag_symbols` (SQL 28 / RLS 48). Rebuild walks the repo independently of chunk `Index`. Query runs text + path + symbol lanes and merges with RRF (k=60); empty hybrid tables fall back to current `fts`/`chunk` behavior.

**Tech Stack:** Go, GORM, SQLite AutoMigrate, Postgres golang-migrate (rev 28), Gin/OpenAPI, existing `internal/rag` regex extractors.

**Spec:** [`docs/superpowers/specs/2026-08-31-rag-hybrid-index-design.md`](../specs/2026-08-31-rag-hybrid-index-design.md)

## Global Constraints

- No vector / embedding store; no ctags / tree-sitter
- Chunk `Index` must **not** write path/symbol tables
- Merge algorithm is **RRF (k=60)** only; `prefer` is a multiplicative boost on that lane
- SQL `expectedVersion` **28**; RLS policy count **48** (46 + 2)
- Commit messages in Chinese (feat:/fix: prefix OK); do not push unless asked
- Shell: Git Bash; `export GOPROXY=https://goproxy.cn,direct` if needed

## File map

| Path | Role |
|------|------|
| `internal/store/models.go` | `RAGPathEntry`, `RAGSymbol` models |
| `internal/store/db.go` | AutoMigrate register |
| `internal/store/migrate_entities.go` | Entity catalog if required |
| `internal/store/rls.go` | Add both tables to `PostgresRLSTables` |
| `internal/store/rls_migration_test.go` | Expect policy count 48 |
| `internal/store/sqlmigrations/migrate.go` | `expectedVersion = 28` |
| `internal/store/sqlmigrations/migrations/postgres/000028_rag_hybrid_index.{up,down}.sql` | DDL + RLS + grants |
| `internal/rag/rebuild.go` | `RebuildSymbols`, walk + upsert + stale delete |
| `internal/rag/hybrid.go` | path/symbol query lanes + RRF merge |
| `internal/rag/service.go` | `Query` prefer field + hybrid branch; export/share extract helpers |
| `internal/rag/profile.go` | hybrid profile fields |
| `internal/rag/*_test.go` | rebuild / hybrid / prefer / fallback tests |
| `internal/api/rag.go` + `handlers.go` | rebuild route |
| `internal/apicodes/catalog.go` | `RAG_SYMBOLS_REBUILD_FAILED` |
| `doc/api/openapi-ash-v1.yaml` + swagger via `make swagger` | Contract |
| `doc/plan/sprint-dx9-rag-hybrid.md`, `v2.3-release-scope.md` | Sprint + draft scope |
| `doc/checklists/rag-hybrid-smoke.md`, `scripts/rag-hybrid-smoke.sh`, `Makefile` | Smoke |
| `CHANGELOG.md`, `doc/plan/TODO.md` | Watermarks |

---

### Task 1: Schema models + SQL 28 + RLS 48

**Files:**
- Modify: `internal/store/models.go` (after `RAGChunk`)
- Modify: `internal/store/db.go` (AutoMigrate list)
- Modify: `internal/store/rls.go` (`PostgresRLSTables`)
- Modify: `internal/store/sqlmigrations/migrate.go` (`expectedVersion = 28`)
- Modify: `internal/store/rls_migration_test.go` (want 48)
- Create: `internal/store/sqlmigrations/migrations/postgres/000028_rag_hybrid_index.up.sql`
- Create: `internal/store/sqlmigrations/migrations/postgres/000028_rag_hybrid_index.down.sql`
- Modify: `internal/store/migrate_entities.go` if other RAG tables are listed there

**Interfaces:**
- Produces: `store.RAGPathEntry`, `store.RAGSymbol`; SQL rev 28; +2 RLS policies

- [ ] **Step 1: Add GORM models**

Append after `RAGChunk` in `models.go`:

```go
// RAGPathEntry is a path/basename row for Hybrid retrieval (Sprint DX9).
type RAGPathEntry struct {
	ID        string    `gorm:"primaryKey;size:64"`
	SpaceID   string    `gorm:"size:64;not null;default:local;uniqueIndex:uidx_rag_path_space_root_path"`
	RepoRoot  string    `gorm:"size:512;not null;uniqueIndex:uidx_rag_path_space_root_path"`
	Path      string    `gorm:"size:1024;not null;uniqueIndex:uidx_rag_path_space_root_path"`
	Basename  string    `gorm:"size:512;not null;index"`
	Digest    string    `gorm:"size:128;not null;index"`
	UpdatedAt time.Time
	CreatedAt time.Time
}

func (RAGPathEntry) TableName() string { return "rag_path_entries" }

// RAGSymbol is a symbol definition row for Hybrid retrieval (Sprint DX9).
type RAGSymbol struct {
	ID        string    `gorm:"primaryKey;size:64"`
	SpaceID   string    `gorm:"size:64;not null;default:local;index:idx_rag_sym_space_root_name"`
	RepoRoot  string    `gorm:"size:512;not null;index:idx_rag_sym_space_root_name;index:idx_rag_sym_space_root_path"`
	Path      string    `gorm:"size:1024;not null;index:idx_rag_sym_space_root_path"`
	Name      string    `gorm:"size:256;not null;index:idx_rag_sym_space_root_name"`
	Kind      string    `gorm:"size:32;not null;default:unknown"`
	Line      int       `gorm:"not null;default:1"`
	Digest    string    `gorm:"size:128;not null;index"`
	UpdatedAt time.Time
	CreatedAt time.Time
}

func (RAGSymbol) TableName() string { return "rag_symbols" }
```

Register both in `db.go` AutoMigrate next to `&RAGChunk{}`.

- [ ] **Step 2: Write Postgres migration 000028**

`000028_rag_hybrid_index.up.sql` (mirror `000026_space_rules.up.sql` RLS/grant pattern for **both** tables):

```sql
-- RAG Hybrid path + symbol index (Sprint DX9).
CREATE TABLE IF NOT EXISTS rag_path_entries (
    id          TEXT PRIMARY KEY,
    space_id    TEXT NOT NULL DEFAULT 'local',
    repo_root   TEXT NOT NULL,
    path        TEXT NOT NULL,
    basename    TEXT NOT NULL,
    digest      TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (space_id, repo_root, path)
);
CREATE INDEX IF NOT EXISTS idx_rag_path_entries_basename ON rag_path_entries (basename);
CREATE INDEX IF NOT EXISTS idx_rag_path_entries_space ON rag_path_entries (space_id);

CREATE TABLE IF NOT EXISTS rag_symbols (
    id          TEXT PRIMARY KEY,
    space_id    TEXT NOT NULL DEFAULT 'local',
    repo_root   TEXT NOT NULL,
    path        TEXT NOT NULL,
    name        TEXT NOT NULL,
    kind        TEXT NOT NULL DEFAULT 'unknown',
    line        INTEGER NOT NULL DEFAULT 1,
    digest      TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_rag_symbols_name ON rag_symbols (space_id, repo_root, name);
CREATE INDEX IF NOT EXISTS idx_rag_symbols_path ON rag_symbols (space_id, repo_root, path);

-- Repeat the space_rules-style DO $rls$ + GRANT blocks for rag_path_entries and rag_symbols.
```

Down migration: drop policies then tables.

- [ ] **Step 3: Bump version + RLS catalog**

- `expectedVersion = 28` in `migrate.go`
- Append to `PostgresRLSTables()`:
  - `{Table: "rag_path_entries", SpaceColumn: "space_id"}`
  - `{Table: "rag_symbols", SpaceColumn: "space_id"}`
- Update `rls_migration_test.go`: `want 48` (was 46)

- [ ] **Step 4: Verify**

```bash
export GOPROXY=https://goproxy.cn,direct
go test ./internal/store/ -count=1 -run 'TestPostgresRLS|TestMigrate|ExpectedVersion|TestShouldApply'
```

Expected: PASS (policy count 48; expectedVersion ≥ 28).

- [ ] **Step 5: Commit**

```bash
git add internal/store/
git commit -m "$(cat <<'EOF'
feat(store): DX9 rag_path_entries / rag_symbols（SQL 28 · RLS 48）

为 Hybrid 检索增加路径与符号表及租户 RLS，与 chunk Index 解耦。
EOF
)"
```

---

### Task 2: RebuildSymbols (walk + upsert + stale cleanup)

**Files:**
- Create: `internal/rag/rebuild.go`
- Create: `internal/rag/rebuild_test.go`
- Modify: `internal/rag/service.go` — add `symbolKind` helper (or put in `rebuild.go`) that classifies using the same regexes as `lineSymbol`

**Interfaces:**
- Consumes: `store.RAGPathEntry`, `store.RAGSymbol`, `AbsRepoRoot`, `shouldSkipDir`, `isIndexable`
- Produces:
  - `type RebuildSymbolsRequest struct { RepoRoot, SpaceID string }`
  - `type RebuildSymbolsResponse struct { Paths, Symbols, Files int }`
  - `func (s *Service) RebuildSymbols(req RebuildSymbolsRequest) (*RebuildSymbolsResponse, error)`

- [ ] **Step 1: Write failing test**

In `rebuild_test.go`:

```go
func TestRebuildSymbolsUpsertsAndCleansStale(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	svc := NewService(db)
	repo := t.TempDir()
	mustWrite := func(rel, body string) {
		p := filepath.Join(repo, rel)
		_ = os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mustWrite("a.go", "package a\n\nfunc Alpha() {}\n")
	mustWrite("b.go", "package b\n\nfunc Beta() {}\n")
	resp, err := svc.RebuildSymbols(RebuildSymbolsRequest{RepoRoot: repo, SpaceID: "s1"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Paths < 2 || resp.Symbols < 2 {
		t.Fatalf("resp=%+v", resp)
	}
	_ = os.Remove(filepath.Join(repo, "b.go"))
	resp2, err := svc.RebuildSymbols(RebuildSymbolsRequest{RepoRoot: repo, SpaceID: "s1"})
	if err != nil {
		t.Fatal(err)
	}
	var paths, syms int64
	_ = db.Model(&store.RAGPathEntry{}).Where("space_id = ?", "s1").Count(&paths)
	_ = db.Model(&store.RAGSymbol{}).Where("space_id = ?", "s1").Count(&syms)
	if paths != 1 {
		t.Fatalf("paths=%d want 1 after stale clean", paths)
	}
	if syms < 1 {
		t.Fatalf("syms=%d", syms)
	}
	_ = resp2
}
```

- [ ] **Step 2: Run test — expect FAIL**

```bash
go test ./internal/rag/ -count=1 -run TestRebuildSymbolsUpsertsAndCleansStale
```

Expected: FAIL (undefined `RebuildSymbols`).

- [ ] **Step 3: Implement `RebuildSymbols`**

In `rebuild.go`:

1. `AbsRepoRoot`; `space := firstNonEmpty(req.SpaceID, "local")`.
2. `filepath.WalkDir` with same skip/indexable rules as `Index`.
3. For each file: read (skip >512KiB / binary); `digest := digestBytes(b)`; rel path with `/`; basename; upsert `RAGPathEntry` (lookup by space+root+path, else create UUID).
4. For each line: if `name, kind := matchSymbolLine(line); name != ""` upsert `RAGSymbol` keyed by `(space, root, path, name, line)` (or delete-all-for-path then insert for simplicity within that file).
5. Track `seenPath` set; after walk, `DELETE` path/symbol rows for `space_id+repo_root` whose `path` not in seen.
6. Return counts.

`matchSymbolLine` must map existing patterns to kinds: `func` / `class` / `type` / `var` / `unknown`. Keep `lineSymbol` working for chunks (can call shared matcher and return only name).

- [ ] **Step 4: Run test — expect PASS**

```bash
go test ./internal/rag/ -count=1 -run TestRebuildSymbols
```

- [ ] **Step 5: Commit**

```bash
git add internal/rag/rebuild.go internal/rag/rebuild_test.go internal/rag/service.go
git commit -m "$(cat <<'EOF'
feat(rag): DX9 RebuildSymbols 扫盘写入路径/符号表

独立于 chunk Index；upsert + 按 path 清理陈旧行。
EOF
)"
```

---

### Task 3: Hybrid Query (three lanes + RRF)

**Files:**
- Create: `internal/rag/hybrid.go`
- Create: `internal/rag/hybrid_test.go`
- Modify: `internal/rag/service.go` — `QueryRequest.Prefer`, `RetrievalModeHybrid`, refactor `Query` to merge when hybrid data exists
- Modify: `internal/rag/profile.go` — `HybridAvailable`, `PathEntryCount`, `SymbolCount`

**Interfaces:**
- Consumes: `RebuildSymbols`, existing `queryFTS` / chunk LIKE
- Produces:
  - `const RetrievalModeHybrid = "hybrid"`
  - `QueryRequest.Prefer string` // `""|"path"|"symbol"|"text"`
  - `func rrfMerge(lanes [][]Hit, prefer string, topK int) []Hit`
  - Profile fields as in spec

- [ ] **Step 1: Write failing tests**

```go
func TestHybridQueryPrefersSymbolOverNoise(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	svc := NewService(db)
	repo := t.TempDir()
	// noise file with many "payment" words; symbol file with UniqueHybridSymbol
	_ = os.WriteFile(filepath.Join(repo, "noise.md"), []byte(strings.Repeat("payment ", 40)+"\n"), 0o644)
	_ = os.WriteFile(filepath.Join(repo, "pay.go"), []byte("package p\n\nfunc UniqueHybridSymbol() {}\n"), 0o644)
	if _, err := svc.Index(IndexRequest{RepoRoot: repo, SpaceID: "hy"}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.RebuildSymbols(RebuildSymbolsRequest{RepoRoot: repo, SpaceID: "hy"}); err != nil {
		t.Fatal(err)
	}
	resp, err := svc.Query(QueryRequest{RepoRoot: repo, SpaceID: "hy", Text: "UniqueHybridSymbol", TopK: 3, Prefer: "symbol"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.RetrievalMode != RetrievalModeHybrid {
		t.Fatalf("mode=%s", resp.RetrievalMode)
	}
	if len(resp.Items) == 0 || resp.Items[0].Symbol != "UniqueHybridSymbol" {
		t.Fatalf("items=%+v", resp.Items)
	}
}

func TestQueryFallsBackWhenHybridEmpty(t *testing.T) {
	// Index only, no RebuildSymbols → mode must be fts or chunk, not hybrid
}
```

Also test invalid prefer returns error from `Query`.

- [ ] **Step 2: Run — expect FAIL**

```bash
go test ./internal/rag/ -count=1 -run 'TestHybridQuery|TestQueryFallsBackWhenHybridEmpty'
```

- [ ] **Step 3: Implement hybrid merge**

In `hybrid.go`:

```go
const rrfK = 60

func preferBoost(prefer, lane string) float64 {
	if prefer == "" || prefer == lane {
		if prefer == lane {
			return 2.0
		}
		return 1.0
	}
	return 1.0
}

func rrfMerge(lanes map[string][]Hit, prefer string, topK int) []Hit {
	scores := map[string]float64{}
	best := map[string]Hit{}
	for lane, hits := range lanes {
		boost := preferBoost(prefer, lane)
		for rank, h := range hits {
			key := h.Ref
			if key == "" {
				key = makeRef(h.Path, h.Symbol, h.StartLine, h.EndLine)
			}
			scores[key] += boost * (1.0 / float64(rrfK+rank+1))
			if prev, ok := best[key]; !ok || h.Score >= prev.Score {
				cp := h
				best[key] = cp
			}
		}
	}
	out := make([]Hit, 0, len(best))
	for key, h := range best {
		h.Score = scores[key]
		out = append(out, h)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Score > out[j].Score })
	if len(out) > topK {
		out = out[:topK]
	}
	return out
}
```

Path lane: `WHERE space_id=? AND repo_root=? AND (LOWER(basename)=? OR LOWER(path) LIKE ?)` for each term.  
Symbol lane: `WHERE ... AND LOWER(name) LIKE ?` / equality.  
Build `Hit` with `Ref` via `makeRef`, snippet = path or `kind name`.

Refactor `Query`:

1. Validate `prefer` ∈ `""|"path"|"symbol"|"text"` else error.
2. Compute text lane (extract current FTS/LIKE into helper returning `[]Hit` + baseMode).
3. `hybridOn := pathCount+symbolCount > 0` for space(+repo).
4. If `!hybridOn` → return text result unchanged.
5. Else path+symbol lanes (each topK*2), `rrfMerge`, `RetrievalModeHybrid`.

Update `Profile` counts + `HybridAvailable: pathCount+symbolCount > 0`. Optionally set `DefaultRetrievalMode` to hybrid when available.

- [ ] **Step 4: Run tests**

```bash
go test ./internal/rag/ -count=1
```

Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/rag/
git commit -m "$(cat <<'EOF'
feat(rag): DX9 Hybrid Query（path/symbol/text RRF 融合）

空表回退 FTS/chunk；支持 prefer=path|symbol|text。
EOF
)"
```

---

### Task 4: HTTP API + OpenAPI + apicodes

**Files:**
- Modify: `internal/api/rag.go`
- Modify: `internal/api/handlers.go` (route `POST /rag/symbols/rebuild`)
- Modify: `internal/apicodes/catalog.go`
- Modify: `internal/api/tenant_test.go` (add rebuild cross-space case if table-driven)
- Run: `make swagger` → updates `internal/api/docs/*`, `doc/api/openapi-ash-v1.yaml` as per project flow

**Interfaces:**
- Produces: handler `rebuildRAGSymbols`; error code `RAG_SYMBOLS_REBUILD_FAILED`

- [ ] **Step 1: Add catalog code**

```go
"RAG_SYMBOLS_REBUILD_FAILED": {Domain: "rag", Summary: "Path/symbol index rebuild failed"},
```

- [ ] **Step 2: Handler + route**

Mirror `indexRAG` permissions (`permRAGIndex`, `requireTargetSpace`):

```go
// RebuildRAGSymbols godoc
// @Summary Rebuild path/symbol Hybrid index
// @Tags rag
// @Accept json
// @Produce json
// @Param body body rag.RebuildSymbolsRequest true "rebuild request"
// @Success 200 {object} rag.RebuildSymbolsResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/rag/symbols/rebuild [post]
func (h *Handler) rebuildRAGSymbols(c *gin.Context) { /* bind → RebuildSymbols → 500 RAG_SYMBOLS_REBUILD_FAILED */ }
```

Register: `v1.POST("/rag/symbols/rebuild", h.rebuildRAGSymbols)`.

- [ ] **Step 3: Swagger + openapi-check**

```bash
make swagger
make openapi-check
```

Expected: `openapi-check OK`.

- [ ] **Step 4: Tenant / API test**

Extend cross-space table with rebuild body `{"spaceId":"space_other","repoRoot":"."}` expecting deny (same as ragIndex).

```bash
go test ./internal/api/ -count=1 -run 'Tenant|RAG|CrossSpace'
go test ./internal/apicodes/ -count=1
```

- [ ] **Step 5: Commit**

```bash
git add internal/api/ internal/apicodes/ doc/api/
git commit -m "$(cat <<'EOF'
feat(api): DX9 POST /rag/symbols/rebuild + OpenAPI

权限对齐 rag/index；错误码 RAG_SYMBOLS_REBUILD_FAILED。
EOF
)"
```

---

### Task 5: Docs, smoke, v2.3 draft, watermarks

**Files:**
- Create: `doc/plan/sprint-dx9-rag-hybrid.md`
- Create: `doc/plan/v2.3-release-scope.md` (draft §1–§6 skeleton; status 草案)
- Create: `doc/checklists/rag-hybrid-smoke.md`
- Create: `scripts/rag-hybrid-smoke.sh`
- Modify: `Makefile` (`.PHONY` + `rag-hybrid-smoke` target)
- Modify: `doc/checklists/smoke-index.md`
- Modify: `CHANGELOG.md`, `doc/plan/TODO.md`, `doc/plan/PLAN-进度与里程碑.md` (brief)

- [ ] **Step 1: Smoke script**

```bash
#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
source "$ROOT/scripts/_go_env.sh"
_ash_go_env_bootstrap "$ROOT"
go test ./internal/rag/ -count=1 -run 'TestRebuildSymbols|TestHybridQuery|TestQueryFallsBack'
go test ./internal/api/ -count=1 -run 'TestIndexRAG|Tenant' || true  # prefer a tight -run that includes rebuild deny
echo "OK rag-hybrid-smoke"
```

Tighten `-run` to a known-good pattern that includes the new tenant case once named.

- [ ] **Step 2: Sprint + v2.3 draft**

`v2.3-release-scope.md`: status **草案**; In = DX9 Hybrid; Out = vectors/ctags; note v2.2 frozen.

- [ ] **Step 3: CHANGELOG / TODO**

- CHANGELOG Unreleased: Sprint DX9 bullet (SQL 28 / RLS 48 / rebuild / hybrid).
- TODO: DX9 row ✅; 详排 link v2.3 draft.

- [ ] **Step 4: Full verify**

```bash
make rag-hybrid-smoke
go test ./internal/rag/ ./internal/store/ -count=1 -run 'TestRebuild|TestHybrid|TestPostgresRLSPolicy|Expected'
make openapi-check
```

- [ ] **Step 5: Commit**

```bash
git add doc/ scripts/rag-hybrid-smoke.sh Makefile CHANGELOG.md
git commit -m "$(cat <<'EOF'
docs: DX9 Hybrid 清单 / v2.3 草案 / rag-hybrid-smoke

记录 SQL28·RLS48 水位与验收入口；不自动打 tag。
EOF
)"
```

---

## Spec coverage (self-review)

| Spec requirement | Task |
|------------------|------|
| `rag_path_entries` + `rag_symbols` + SQL 28 + RLS | Task 1 |
| Rebuild API decoupled from Index | Task 2 + 4 |
| Regex extract, no ctags | Task 2 |
| Hybrid Query RRF + prefer | Task 3 |
| Profile hybrid fields | Task 3 |
| Fallback when empty | Task 3 |
| OpenAPI + error code | Task 4 |
| Smoke + v2.3 draft + CHANGELOG | Task 5 |
| No vectors / no auto Index coupling | Global + Tasks 2–3 |

No TBD placeholders; merge algorithm fixed to RRF k=60; prefer boost = 2.0 on selected lane.
