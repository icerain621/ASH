# Waker Duties Ledger (DX12) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `waker_duties` / `waker_duty_runs` ledger, migrate `stale_run` into scheduled duties, expose `/waker/status` + duties list/run APIs, and drive the Worker ticker from the ledger (report/flag only).

**Architecture:** SQL rev **29** / RLS **50**. Default per-space `stale_run` duty is ensured on first status/ticker touch. Existing `GET /waker/queue` and `POST /waker/sweep` keep working; cancel gates unchanged. DX13/DX14 (doctor/kpi probes + UI + freeze) are **out of this plan** — see spec.

**Tech Stack:** Go, GORM, SQLite AutoMigrate, Postgres golang-migrate, Gin/OpenAPI, existing `internal/waker`.

**Spec:** [`docs/superpowers/specs/2026-09-01-waker-duties-design.md`](../specs/2026-09-01-waker-duties-design.md)

## Global Constraints

- Background / duty runs **never** cancel (DX7 gates only on explicit `/sweep` cancel)
- `doctor_subset` / `kpi_drift` probes are **DX13** — DX12 may store `kind` enum but only implement `stale_run`
- SQL `expectedVersion` **29**; RLS policy count **48 → 50** (+2 tables)
- Commit messages in Chinese (`feat:` / `fix:` OK); do not push unless asked
- Shell: Git Bash; `export GOPROXY=https://goproxy.cn,direct` if needed
- Do not reopen v2.3 frozen Hybrid scope

## File map

| Path | Role |
|------|------|
| `internal/store/models.go` | `WakerDuty`, `WakerDutyRun` |
| `internal/store/db.go` | AutoMigrate |
| `internal/store/rls.go` | `PostgresRLSTables` entries |
| `internal/store/rls_migration_test.go` | expect **50** |
| `internal/store/sqlmigrations/migrate.go` | `expectedVersion = 29` |
| `internal/store/sqlmigrations/migrations/postgres/000029_waker_duties.{up,down}.sql` | DDL + RLS + grants |
| `internal/store/migrate_entities.go` | entity catalog if listed |
| `internal/waker/duty.go` | ensure/list/status/run duty + write duty_runs |
| `internal/waker/service.go` | Item.Kind; StartBackground → due duties |
| `internal/waker/duty_test.go` | ledger + ensure + run tests |
| `internal/api/waker.go` | status / duties handlers |
| `internal/api/handlers.go` | routes |
| `internal/api/waker_test.go` | HTTP coverage |
| `internal/apicodes/catalog.go` | status/duty error codes |
| `doc/api/openapi-ash-v1.yaml` + `make swagger` | contract |
| `scripts/waker-smoke.sh`, `doc/checklists/waker-smoke.md` | smoke extends |
| `doc/plan/sprint-dx12-waker-duties.md`, `doc/plan/v2.4-release-scope.md` | sprint + draft scope |
| `CHANGELOG.md`, `doc/plan/TODO.md` | watermarks |

---

### Task 1: Schema models + SQL 29 + RLS 50

**Files:**
- Modify: `internal/store/models.go` (after RAG models or near ops tables)
- Modify: `internal/store/db.go`
- Modify: `internal/store/rls.go`
- Modify: `internal/store/sqlmigrations/migrate.go`
- Modify: `internal/store/rls_migration_test.go`
- Create: `internal/store/sqlmigrations/migrations/postgres/000029_waker_duties.up.sql`
- Create: `internal/store/sqlmigrations/migrations/postgres/000029_waker_duties.down.sql`
- Modify: `internal/store/migrate_entities.go` if applicable

**Interfaces:**
- Produces: `store.WakerDuty`, `store.WakerDutyRun`; SQL rev 29; RLS count 50

- [ ] **Step 1: Write failing watermark test expectation**

In `rls_migration_test.go`, change expected policy count from `48` to `50`. Run:

```bash
export GOPROXY=https://goproxy.cn,direct
go test ./internal/store/ -run 'TestPostgresRLS|TestExpectedVersion|TestMigrate' -count=1
```

Expected: FAIL (count still 48 / version still 28).

- [ ] **Step 2: Add GORM models**

```go
// WakerDuty is a scheduled continuous-duty definition (Sprint DX12).
type WakerDuty struct {
	ID         string `gorm:"primaryKey;size:64"`
	SpaceID    string `gorm:"size:64;not null;default:local;uniqueIndex:uidx_waker_duty_space_kind"`
	Kind       string `gorm:"size:32;not null;uniqueIndex:uidx_waker_duty_space_kind"` // stale_run | doctor_subset | kpi_drift
	Enabled    bool   `gorm:"not null;default:true"`
	IntervalMs int64  `gorm:"not null;default:300000"` // 5m
	ConfigJSON string `gorm:"type:text;not null;default:'{}'"`
	NextRunAt  time.Time
	UpdatedAt  time.Time
	CreatedAt  time.Time
}

func (WakerDuty) TableName() string { return "waker_duties" }

// WakerDutyRun is one execution summary for a duty.
type WakerDutyRun struct {
	ID         string `gorm:"primaryKey;size:64"`
	SpaceID    string `gorm:"size:64;not null;default:local;index:idx_waker_duty_runs_space_started"`
	DutyID     string `gorm:"size:64;not null;index:idx_waker_duty_runs_duty_started"`
	Kind       string `gorm:"size:32;not null"`
	Status     string `gorm:"size:16;not null"` // ok | failed | skipped
	Matched    int
	Flagged    int
	Canceled   int
	Summary    string    `gorm:"size:512"`
	StartedAt  time.Time `gorm:"index:idx_waker_duty_runs_space_started;index:idx_waker_duty_runs_duty_started"`
	FinishedAt time.Time
}

func (WakerDutyRun) TableName() string { return "waker_duty_runs" }
```

Register both in `db.go` AutoMigrate.

- [ ] **Step 3: Postgres migration 000029**

`000029_waker_duties.up.sql` (mirror Hybrid RLS/grants style):

```sql
-- Waker duties ledger (Sprint DX12).
CREATE TABLE IF NOT EXISTS waker_duties (
    id           TEXT PRIMARY KEY,
    space_id     TEXT NOT NULL DEFAULT 'local',
    kind         TEXT NOT NULL,
    enabled      BOOLEAN NOT NULL DEFAULT TRUE,
    interval_ms  BIGINT NOT NULL DEFAULT 300000,
    config_json  TEXT NOT NULL DEFAULT '{}',
    next_run_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (space_id, kind)
);
CREATE INDEX IF NOT EXISTS idx_waker_duties_next ON waker_duties (enabled, next_run_at);

CREATE TABLE IF NOT EXISTS waker_duty_runs (
    id          TEXT PRIMARY KEY,
    space_id    TEXT NOT NULL DEFAULT 'local',
    duty_id     TEXT NOT NULL,
    kind        TEXT NOT NULL,
    status      TEXT NOT NULL,
    matched     INTEGER NOT NULL DEFAULT 0,
    flagged     INTEGER NOT NULL DEFAULT 0,
    canceled    INTEGER NOT NULL DEFAULT 0,
    summary     TEXT NOT NULL DEFAULT '',
    started_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_waker_duty_runs_space_started ON waker_duty_runs (space_id, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_waker_duty_runs_duty_started ON waker_duty_runs (duty_id, started_at DESC);

-- RLS + grants: duplicate the two DO $rls$ / GRANT blocks from 000028 for these table names.
```

`down.sql`: drop policies then tables (duty_runs first).

Set `expectedVersion = 29` in `migrate.go`. Append `{Table: "waker_duties", ...}` and `{Table: "waker_duty_runs", ...}` to `PostgresRLSTables()`.

- [ ] **Step 4: Re-run store tests**

```bash
go test ./internal/store/ -run 'TestPostgresRLS|TestExpectedVersion|TestEmbeddedMigrations' -count=1
```

Expected: PASS (policy 50, version 29, embedded up count matches).

- [ ] **Step 5: Commit** (only if user asked)

```bash
git add internal/store/
git commit -m "$(cat <<'EOF'
feat(store): DX12 Waker duties 表 SQL 29 / RLS 50

为持续职责账本落地 schema，供调度与审计使用。
EOF
)"
```

---

### Task 2: Duty ensure + RunStaleDuty + Status (package)

**Files:**
- Create: `internal/waker/duty.go`
- Modify: `internal/waker/service.go` (`Item.Kind` string `json:"kind,omitempty"`)
- Create: `internal/waker/duty_test.go`
- Modify: `internal/waker/service_test.go` if queue assertions need Kind

**Interfaces:**
- Consumes: `store.WakerDuty`, `store.WakerDutyRun`, existing `listStale` / `Sweep`
- Produces:
  - `const KindStaleRun = "stale_run"`
  - `func (s *Service) EnsureStaleRunDuty(spaceID string) (store.WakerDuty, error)`
  - `func (s *Service) ListDuties(spaceID string) ([]store.WakerDuty, error)`
  - `func (s *Service) Status(spaceID string, recent int) (StatusResponse, error)`
  - `func (s *Service) RunDuty(dutyID string, dryRun bool) (SweepResponse, error)` — stale_run only in DX12
  - `func (s *Service) RunDueDuties(now time.Time) (int, error)` — background; never cancel
  - types `StatusResponse`, `DutyStatusView`, `DutyRunView`

- [ ] **Step 1: Failing tests**

```go
func TestEnsureStaleRunDutyIdempotent(t *testing.T) {
	db := openTestDB(t) // reuse existing waker test helper pattern
	svc := NewService(db)
	a, err := svc.EnsureStaleRunDuty("local")
	if err != nil { t.Fatal(err) }
	b, err := svc.EnsureStaleRunDuty("local")
	if err != nil { t.Fatal(err) }
	if a.ID == "" || a.ID != b.ID || a.Kind != KindStaleRun {
		t.Fatalf("idempotent ensure failed: %+v vs %+v", a, b)
	}
}

func TestRunDueDutiesWritesDutyRun(t *testing.T) {
	t.Setenv("ASH_WAKER_RUN_TTL", "1h")
	db := openTestDB(t)
	seedStaleRunningRun(t, db) // same fixture as Queue tests
	svc := NewService(db)
	if _, err := svc.EnsureStaleRunDuty("local"); err != nil { t.Fatal(err) }
	// force next_run_at in the past via DB update
	n, err := svc.RunDueDuties(time.Now().UTC())
	if err != nil || n < 1 { t.Fatalf("n=%d err=%v", n, err) }
	st, err := svc.Status("local", 5)
	if err != nil { t.Fatal(err) }
	if len(st.RecentRuns) < 1 || st.RecentRuns[0].Kind != KindStaleRun {
		t.Fatalf("want duty run: %+v", st)
	}
}
```

Adapt helpers to whatever `service_test.go` already uses (`store.Open` temp SQLite).

Run: `go test ./internal/waker/ -run 'TestEnsureStaleRunDuty|TestRunDueDuties' -count=1`  
Expected: FAIL (undefined).

- [ ] **Step 2: Implement `duty.go`**

Key behaviors:

- `EnsureStaleRunDuty`: upsert unique `(space_id, kind=stale_run)`; default `IntervalMs` from `ASH_WAKER_INTERVAL` if parseable else `300000`; `NextRunAt=now`; `ConfigJSON="{}"`.
- IDs: use existing project ID helper if any (`store`/uuid); otherwise `fmt.Sprintf("wd_%d", time.Now().UnixNano())` consistent with nearby packages.
- `RunDueDuties`: list `enabled AND next_run_at <= now`; for each `kind==stale_run` call internal report sweep (`dryRun=false`, `action=report` only — **flag path, never cancel**); insert `WakerDutyRun`; set `next_run_at = now + interval`; skip unknown kinds with `status=skipped` (DX13 placeholders).
- `Status`: ensure stale duty for space; return duties + last N runs + `AllowCancel` bool + ticker hint from env.
- `RunDuty`: load by id; if kind != stale_run return error `unsupported duty kind`; else Sweep report and persist duty_run.
- Queue `Item`: set `Kind: KindStaleRun` in `listStale`.

- [ ] **Step 3: Wire `StartBackground`**

Replace body of ticker `run` with:

```go
if _, err := svc.EnsureStaleRunDuty("local"); err != nil {
    log.Printf("waker: ensure: %v", err)
}
n, err := svc.RunDueDuties(time.Now().UTC())
if err != nil {
    log.Printf("waker: due: %v", err)
    return
}
if n > 0 {
    log.Printf("waker: ran %d due dut(ies)", n)
}
```

Also ensure duties for other known spaces if an existing helper lists spaces; if none, **document** DX12 limitation: background ensures `local` only; HTTP `Status(spaceId)` ensures that space. (Optional stretch: distinct `space_id` from recent runs — only if cheap.)

- [ ] **Step 4: Pass tests**

```bash
go test ./internal/waker/ -count=1
```

Expected: PASS (including existing cancel tests).

- [ ] **Step 5: Commit** (if asked)

```bash
git commit -m "$(cat <<'EOF'
feat(waker): DX12 stale_run 职责账本与到期调度

将后台巡检迁到 waker_duties，并落库 duty_runs 审计行。
EOF
)"
```

---

### Task 3: HTTP API + OpenAPI

**Files:**
- Modify: `internal/api/waker.go`
- Modify: `internal/api/handlers.go` (routes)
- Modify: `internal/api/waker_test.go`
- Modify: `internal/apicodes/catalog.go`
- Modify: `doc/api/openapi-ash-v1.yaml`
- Regen: `make swagger` → `internal/api/docs/*`

**Interfaces:**
- `GET /api/v1/waker/status?spaceId=&recent=`
- `GET /api/v1/waker/duties?spaceId=`
- `POST /api/v1/waker/duties/:id/run` body `{ "dryRun": true }`

- [ ] **Step 1: Failing API test**

```go
func TestWakerStatusAndDuties(t *testing.T) {
	// boot test server like TestWakerQueueAndSweep
	req := httptest.NewRequest(http.MethodGet, "/api/v1/waker/status?spaceId=local&recent=3", nil)
	// expect 200, duties len>=1, kind stale_run
	list := httptest.NewRequest(http.MethodGet, "/api/v1/waker/duties?spaceId=local", nil)
	// expect 200
}
```

Run until FAIL.

- [ ] **Step 2: Handlers + codes**

Add catalog:

- `WAKER_STATUS_FAILED`
- `WAKER_DUTIES_FAILED`
- `WAKER_DUTY_RUN_FAILED`

Swagger comments on handlers; register routes next to existing waker routes.

- [ ] **Step 3: OpenAPI + swagger**

Update `doc/api/openapi-ash-v1.yaml` paths + schemas (`WakerStatusResponse`, duty models). Then:

```bash
make swagger
bash scripts/openapi-check.sh
```

Expected: `openapi-check OK`.

- [ ] **Step 4: API tests green**

```bash
go test ./internal/api/ -run 'TestWaker' -count=1
```

- [ ] **Step 5: Commit** (if asked)

```bash
git commit -m "$(cat <<'EOF'
feat(api): DX12 /waker/status 与 duties 运行接口

暴露职责账本只读与手动 report 触发，便于运维与后续 UI。
EOF
)"
```

---

### Task 4: Smoke + plan docs watermark

**Files:**
- Modify: `scripts/waker-smoke.sh`
- Modify: `doc/checklists/waker-smoke.md`
- Create: `doc/plan/sprint-dx12-waker-duties.md`
- Create: `doc/plan/v2.4-release-scope.md` (draft, **未冻结**)
- Modify: `doc/plan/TODO.md`, `CHANGELOG.md`, `doc/checklists/smoke-index.md` if needed
- Modify: `doc/plan/PLAN-进度与里程碑.md` one-line watermark (v2.4 DX12 进行中)

- [ ] **Step 1: Extend smoke**

After package tests, assert status in-process is enough; optionally curl status when `ASH_WORKER_URL` set:

```bash
go test ./internal/waker/ ./internal/api/ -count=1 -run 'TestWaker|TestEnsure|TestRunDue|TestQueue'
# live:
curl -sf "${BASE}/api/v1/waker/status?spaceId=local&recent=3" | head -c 500
```

- [ ] **Step 2: Sprint + v2.4 draft scope**

`v2.4-release-scope.md` status **草案**; In: DX12–DX14 per spec; Out: Landlock, default auto-cancel, vectors.

- [ ] **Step 3: Run smoke**

```bash
make waker-smoke
```

Expected: `OK waker-smoke`.

- [ ] **Step 4: Commit** (if asked)

```bash
git commit -m "$(cat <<'EOF'
docs: DX12 Waker duties sprint 与 v2.4 草案

对齐烟测与范围水位，DX13/UI 仍后续。
EOF
)"
```

---

## Follow-on (not this plan)

| Sprint | Plan later |
|--------|------------|
| DX13 | `doctor_subset` + `kpi_drift` probes |
| DX14 | Observability/Scale UI + `make v2.4-signoff` |

---

## Plan self-review

| Spec (DX12 slice) | Task |
|-------------------|------|
| `waker_duties` / `waker_duty_runs` + SQL 29 / RLS 50 | T1 |
| Ensure default `stale_run`; ticker from ledger; never cancel | T2 |
| `/status`, `/duties`, `/duties/:id/run` + OpenAPI | T3 |
| Smoke + sprint / v2.4 draft docs | T4 |
| DX13/14 UI & probes | Explicitly deferred |

No TBD placeholders; cancel gates untouched; kinds other than `stale_run` skipped until DX13.
