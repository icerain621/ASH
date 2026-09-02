# Waker Multi-duty Probes (DX13) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement opt-in `doctor_subset` and `kpi_drift` probes that write `waker_duty_runs`, merge findings into `GET /waker/queue`, and execute from `RunDueDuties` / `RunDuty` — without auto-seed, cancel, or new schema.

**Architecture:** Add `internal/waker/probes.go` with injectable `DoctorRunner` + DB-backed KPI-17 backlog count. Extend duty scheduler switch; synthesize queue items from recent flagged duty runs. No SQL rev bump.

**Tech Stack:** Go, GORM, existing `internal/waker` ledger (SQL 29 / RLS 50), optional Doctor suite via injected interface.

**Spec:** [`docs/superpowers/specs/2026-09-01-waker-duties-design.md`](../specs/2026-09-01-waker-duties-design.md) · Addendum [`docs/superpowers/specs/2026-09-02-waker-duties-dx13-addendum.md`](../specs/2026-09-02-waker-duties-dx13-addendum.md)

## Global Constraints

- Background / duty runs **never** cancel
- **Do not** auto-seed `doctor_subset` / `kpi_drift` (方案 A)
- Default doctor suite **M4**; default KPI metric **KPI-17** with absolute `threshold` default **50** (align `evolveBacklogTarget`)
- No new SQL migration / RLS count change
- Commit messages in Chinese; do not push unless asked
- Shell: Git Bash; `export GOPROXY=https://goproxy.cn,direct` if needed
- Do not reopen v2.3 Hybrid scope; DX14 UI / `v2.4-signoff` out of this plan

## File map

| Path | Role |
|------|------|
| `internal/waker/probes.go` | Config parse, `runDoctorSubset`, `runKPIDrift`, DoctorRunner interface |
| `internal/waker/duty.go` | Kind constants; Ensure* (opt-in); RunDueDuties/RunDuty switch; Queue merge |
| `internal/waker/service.go` | Service fields for DoctorRunner; Queue extension hook |
| `internal/waker/probes_test.go` | Fake doctor + KPI drift / doctor fail tests |
| `internal/waker/duty_test.go` | Update skip tests → execute; ensure no auto-seed |
| `cmd/worker/main.go` and/or `internal/api/handlers.go` | Wire DoctorRunner when constructing waker (best-effort) |
| `scripts/waker-smoke.sh`, `doc/checklists/waker-smoke.md` | Cover probe tests |
| `doc/plan/sprint-dx13-waker-probes.md`, TODO/CHANGELOG, `v2.4-release-scope.md` watermark | Docs |

---

### Task 1: Probe runners + config (package, TDD)

**Files:**
- Create: `internal/waker/probes.go`
- Create: `internal/waker/probes_test.go`
- Modify: `internal/waker/duty.go` / `service.go` (Service fields only if needed)

**Interfaces:**
- Produces:
  - `const KindDoctorSubset = "doctor_subset"`, `KindKPIDrift = "kpi_drift"`
  - `type DoctorCaseResult struct { ID, Status string }` // Status pass|fail (and treat other as non-fail)
  - `type DoctorReport struct { Cases []DoctorCaseResult }`
  - `type DoctorRunner interface { RunSuite(suite string) (*DoctorReport, error) }`
  - `func (s *Service) WithDoctorRunner(r DoctorRunner) *Service`
  - `func parseDoctorConfig(json string) (suite string, caseIDs []string, err error)`
  - `func parseKPIConfig(json string) (metric string, threshold float64, baseline *float64, mode string, err error)`
  - `func (s *Service) runDoctorSubset(duty store.WakerDuty, dryRun bool) (SweepResponse, error)`
  - `func (s *Service) runKPIDrift(duty store.WakerDuty, dryRun bool) (SweepResponse, error)`

- [ ] **Step 1: Failing tests**

```go
func TestParseDoctorConfigDefaults(t *testing.T) {
	suite, ids, err := parseDoctorConfig("{}")
	if err != nil || suite != "M4" || len(ids) != 0 {
		t.Fatalf("suite=%q ids=%v err=%v", suite, ids, err)
	}
}

func TestRunDoctorSubsetFlagsFailures(t *testing.T) {
	db := openDutyTestDB(t) // reuse helper from duty_test
	svc := NewService(db).WithDoctorRunner(fakeDoctor{failIDs: []string{"M4-HAR-01"}})
	duty := store.WakerDuty{ID: "wd_d", SpaceID: "local", Kind: KindDoctorSubset,
		ConfigJSON: `{"suite":"M4"}`, Enabled: true, IntervalMs: 300000}
	resp, err := svc.runDoctorSubset(duty, false)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Flagged < 1 || resp.Matched < 1 {
		t.Fatalf("want flagged failures: %+v", resp)
	}
	if !strings.Contains(resp.Summary, "doctor_subset:M4-HAR-01") {
		t.Fatalf("summary=%q", resp.Summary)
	}
}

func TestRunKPIDriftFlagsOverThreshold(t *testing.T) {
	db := openDutyTestDB(t)
	// insert >50 in_review harness or patch rows OR stub via direct count setup
	svc := NewService(db)
	duty := store.WakerDuty{ID: "wd_k", SpaceID: "local", Kind: KindKPIDrift,
		ConfigJSON: `{"metric":"KPI-17","threshold":0}`, Enabled: true, IntervalMs: 300000}
	// With threshold 0, any backlog >=0 with items flags; use threshold 0 and seed 1 in_review
	resp, err := svc.runKPIDrift(duty, false)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Flagged < 1 {
		t.Fatalf("want flag: %+v", resp)
	}
}
```

Adapt seeding to existing store models (`HarnessProfileVersion` / `ScenarioPatchDraft` with `status=in_review`). If models are heavy, prefer a test-only `kpiCounter` hook:

```go
// optional for tests
type KPIBacklogFunc func(spaceID string) (int64, error)
func (s *Service) WithKPIBacklog(fn KPIBacklogFunc) *Service
```

Default implementation queries harness+patch `in_review` counts (same as KPI-17).

Run: `go test ./internal/waker/ -run 'TestParseDoctor|TestRunDoctor|TestRunKPI' -count=1`  
Expected: FAIL (undefined).

- [ ] **Step 2: Implement `probes.go`**

Key rules:

- `runDoctorSubset`: if `DoctorRunner == nil` → return SweepResponse with empty match and treat as **skipped** at caller (or return error `doctor runner unavailable` and caller sets `skipped`). Prefer: return `errDoctorUnavailable` and RunDueDuties sets `status=skipped`.
- Filter cases by `caseIds` if non-empty.
- `Matched` = examined case count; `Flagged` = fail count; `Summary` joins `doctor_subset:<id>` for fails.
- `dryRun=true`: still compute matched/flagged but do not require side effects (there are none besides duty_run persistence at caller).
- `runKPIDrift`: only `KPI-17` in DX13; unknown metric → error / skipped.
- Absolute mode (default): flag when `backlog >= threshold` (threshold default 50).
- `Matched` = int(backlog); `Flagged` = 1 if over else 0; `Summary` = `kpi_drift:KPI-17 backlog=N threshold=T`.
- Optional `baseline`: flag when `backlog >= baseline + threshold` if both set — keep simple: if `baseline` present, flag when `backlog > *baseline + 0` and `backlog >= threshold` OR simply `backlog >= max(threshold, baseline)` — **use:** flag iff `backlog >= threshold` when baseline nil; if baseline set, flag iff `backlog - *baseline >= threshold` (delta from baseline). Document in comments.

- [ ] **Step 3: Pass tests**

```bash
export GOPROXY=https://goproxy.cn,direct
go test ./internal/waker/ -run 'TestParseDoctor|TestRunDoctor|TestRunKPI' -count=1
```

Expected: PASS.

- [ ] **Step 4: Commit** (if asked / SDD authorized)

```bash
git add internal/waker/probes.go internal/waker/probes_test.go internal/waker/duty.go internal/waker/service.go
git commit -m "$(cat <<'EOF'
feat(waker): DX13 doctor_subset 与 kpi_drift 探针实现

可选注入 DoctorRunner；KPI-17 积压超阈仅 flag，不 cancel。
EOF
)"
```

---

### Task 2: Scheduler + Ensure + Queue merge

**Files:**
- Modify: `internal/waker/duty.go`
- Modify: `internal/waker/service.go` (`Queue`)
- Modify: `internal/waker/duty_test.go` (replace skip-as-success expectations)

**Interfaces:**
- `EnsureDoctorSubsetDuty(spaceID string, enabled bool) (store.WakerDuty, error)` — default config M4; **does not** run on Status
- `EnsureKPIDriftDuty(spaceID string, enabled bool) (store.WakerDuty, error)` — default KPI-17/threshold 50
- `RunDuty` / `RunDueDuties` execute both kinds
- `Queue` appends probe items from recent flagged runs

- [ ] **Step 1: Failing tests**

```go
func TestRunDueDutiesExecutesDoctorSubset(t *testing.T) {
	// ensure doctor duty enabled + due; WithDoctorRunner fake fail; RunDueDuties; Status recent run status ok/failed with flagged
}

func TestQueueIncludesDoctorFindings(t *testing.T) {
	// persist a duty_run flagged with summary doctor_subset:M4-HAR-01 OR run probe then Queue
	q, err := svc.Queue("local", "", 50)
	// expect some Item with Kind==KindDoctorSubset and Reason containing doctor_subset:
}

func TestStatusDoesNotAutoSeedDoctorOrKPI(t *testing.T) {
	svc.Status("local", 5)
	list, _ := svc.ListDuties("local")
	for _, d := range list {
		if d.Kind == KindDoctorSubset || d.Kind == KindKPIDrift {
			t.Fatal("must not auto-seed")
		}
	}
}
```

Update `TestRunDueDutiesSkipsUnknownKind` → unknown kinds still skipped; remove kpi_drift from that test (kpi now supported). Use a fake kind `future_probe` for skip test.

- [ ] **Step 2: Implement**

`Queue` algorithm:

1. Existing `listStale` items.
2. Load last 20 `waker_duty_runs` for space where `kind IN (doctor_subset,kpi_drift)` AND `flagged > 0` ordered by `started_at DESC`.
3. Parse `Summary` for tokens `doctor_subset:ID` / `kpi_drift:METRIC` (split by `;` or whitespace); emit one `Item` per token (dedupe by reason).
4. `Item.RunID` may be empty or use duty_run id; `Reason` = token; `Kind` = duty kind; `Status` = `flagged`.

`RunDueDuties` switch:

```go
case KindStaleRun: ...
case KindDoctorSubset: resp, runErr = s.runDoctorSubset(duty, false)
case KindKPIDrift: resp, runErr = s.runKPIDrift(duty, false)
default: skipped
```

If `runErr == errDoctorUnavailable` → status skipped.

- [ ] **Step 3: Full package tests**

```bash
go test ./internal/waker/ -count=1
```

Expected: PASS (including never-cancel).

- [ ] **Step 4: Commit**

```bash
git commit -m "$(cat <<'EOF'
feat(waker): DX13 调度执行探针并合并 queue 发现项

RunDueDuties/RunDuty 支持 doctor/kpi；Status 不自动 seed。
EOF
)"
```

---

### Task 3: Wire DoctorRunner + API smoke coverage

**Files:**
- Modify: `internal/api/handlers.go` and/or `cmd/worker/main.go` — attach DoctorRunner adapter
- Modify: `internal/api/waker_test.go` — optional HTTP queue with probe item
- Modify: `scripts/waker-smoke.sh`, `doc/checklists/waker-smoke.md`
- OpenAPI: only if `Item` docs need `kind` already present — verify swagger comments; regen if touched

**Doctor adapter sketch:**

```go
type doctorSuiteAdapter struct{ svc *doctor.Service }
func (a doctorSuiteAdapter) RunSuite(suite string) (*waker.DoctorReport, error) {
	rep, err := a.svc.RunSuite(suite)
	// map CaseResult ID/Status → DoctorCaseResult
}
```

Wire where Handler already has doctor access — if Handler does not hold doctor.Service, construct adapter in worker when starting waker background **or** leave DoctorRunner nil in API tests and document that Worker wiring is required for live doctor duties. Prefer: add optional set on Handler after NewHandler if doctor is available elsewhere in `cmd/worker`.

Search `doctor.NewService` in `cmd/worker` / `handlers` and attach with minimal churn. If wiring is awkward, **YAGNI**: package tests cover probes with fake; smoke runs package tests; document Worker wire as follow-up in sprint note — but try hard to wire in `cmd/worker` next to `waker.StartBackground`.

- [ ] **Step 1: Wire or document**

If `cmd/worker` has doctor service, set:

```go
wakerSvc := waker.NewService(db).WithDoctorRunner(adapter)
stopWaker := waker.StartBackground(...) // may need StartBackground to accept *Service
```

Check `StartBackground(db, interval)` — if it constructs its own Service, change to `StartBackgroundService(svc, interval)` **or** set runner via package-level — prefer refactor StartBackground to use `NewService(db).WithDoctorRunner(...)` if runner registered on a small setter:

```go
var defaultDoctor DoctorRunner
func SetDefaultDoctorRunner(r DoctorRunner) { defaultDoctor = r }
// NewService copies defaultDoctor
```

Keep it simple: `StartBackground` calls `NewService(db)` then `WithDoctorRunner(defaultDoctor)`.

- [ ] **Step 2: Extend smoke**

```bash
go test ./internal/waker/ ./internal/api/ -count=1 -run 'TestWaker|TestEnsure|TestRunDue|TestQueue|TestRunDoctor|TestRunKPI|TestParse'
```

- [ ] **Step 3: `make waker-smoke`**

Expected: `OK waker-smoke`.

- [ ] **Step 4: Commit**

```bash
git commit -m "$(cat <<'EOF'
feat(waker): DX13 接入 DoctorRunner 与烟测覆盖

Worker 可跑 doctor_subset；包测覆盖 KPI flag 与 queue 合并。
EOF
)"
```

---

### Task 4: Docs watermark

**Files:**
- Create: `doc/plan/sprint-dx13-waker-probes.md`
- Modify: `doc/plan/TODO.md` (DX13 ✅)
- Modify: `CHANGELOG.md`
- Modify: `doc/plan/v2.4-release-scope.md` (DX13 done; still 草案)
- Modify: `doc/plan/PLAN-进度与里程碑.md` one-liner if needed
- Commit addendum + plan under `docs/superpowers/`

- [ ] **Step 1: Write sprint note** — Goal/tasks/exit; Out = DX14 UI

- [ ] **Step 2: Watermarks** — DX13 ✅; next DX14

- [ ] **Step 3: Commit**

```bash
git commit -m "$(cat <<'EOF'
docs: DX13 Waker 多职责探针 sprint 与水位

方案 A：可选启用 doctor_subset/kpi_drift；v2.4 仍为草案。
EOF
)"
```

---

## Follow-on

| Sprint | Work |
|--------|------|
| DX14 | Observability/Scale Waker UI + freeze + `make v2.4-signoff` |

---

## Plan self-review

| Spec / Addendum | Task |
|-----------------|------|
| doctor_subset M4 + caseIds; fail → queue reason | T1–T2 |
| kpi_drift KPI-17 + threshold; flag only | T1–T2 |
| No auto-seed | T2 Status test |
| Never cancel | Existing cancel tests stay green |
| No new SQL | No Task for migrate |
| Smoke + docs | T3–T4 |
| UI / v2.4 freeze | Deferred DX14 |
