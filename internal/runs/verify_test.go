package runs

import (
	"strings"
	"testing"
	"time"

	"github.com/ash-repwiki/ash/internal/events"
	"github.com/ash-repwiki/ash/internal/rules"
	"github.com/ash-repwiki/ash/internal/store"
)

type stubImprove struct {
	calls int
	last  string
}

func (s *stubImprove) DraftFromVerifyFailure(spaceID, runID, stepID, detail string) (string, error) {
	s.calls++
	s.last = stepID + ":" + detail
	return "imp_test", nil
}

func TestVerifyHelpers(t *testing.T) {
	step := rules.Step{Retry: &rules.RetrySpec{MaxAttempts: 3, BackoffMs: 5}}
	if verifyMaxAttempts(step) != 3 {
		t.Fatal("max attempts")
	}
	if verifyBackoff(step) != 5*time.Millisecond {
		t.Fatal("backoff")
	}
	if verifyMaxAttempts(rules.Step{}) != 1 {
		t.Fatal("default attempts")
	}
}

func TestMaybeDraftImprove(t *testing.T) {
	stub := &stubImprove{}
	svc := &Service{improve: stub}
	rec := &store.RunRecord{ID: "run_1", TraceID: "trc_1", SpaceID: "local"}
	svc.maybeDraftImprove(rec, "run_1", "qa.verify", "test.run failed")
	if stub.calls != 1 || !strings.Contains(stub.last, "qa.verify") {
		t.Fatalf("%+v", stub)
	}
}

func TestMaybeProposeReplanBounded(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	now := time.Now().UTC()
	run := store.RunRecord{
		ID: "run_replan", SpaceID: "local", TraceID: "tr", Status: "failed",
		ScenarioName: "x", ScenarioVersion: "1", StartedAt: now, CreatedAt: now, UpdatedAt: now,
	}
	if err := db.Create(&run).Error; err != nil {
		t.Fatal(err)
	}
	stub := &stubImprove{}
	svc := &Service{db: db, events: events.NewService(db), improve: stub}
	step := rules.Step{ID: "qa.verify", Verify: &rules.VerifySpec{OnFail: "replan", MaxReplanTurns: 2, Checks: []rules.ToolChainItem{{Tool: "echo"}}}}
	rec := &run

	if exhausted := svc.maybeProposeReplan(rec, run.ID, step, "first fail"); exhausted {
		t.Fatal("first replan should be allowed")
	}
	if stub.calls != 1 {
		t.Fatalf("improve drafts=%d", stub.calls)
	}
	if exhausted := svc.maybeProposeReplan(rec, run.ID, step, "second fail"); exhausted {
		t.Fatal("second replan should be allowed")
	}
	if exhausted := svc.maybeProposeReplan(rec, run.ID, step, "third fail"); !exhausted {
		t.Fatal("third replan must be exhausted")
	}
	var n int64
	if err := db.Model(&store.RunEvent{}).Where("run_id = ? AND type = ?", run.ID, "harness.replan_proposed").Count(&n).Error; err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("proposed events=%d want 2", n)
	}
	var exhaustedN int64
	_ = db.Model(&store.RunEvent{}).Where("run_id = ? AND type = ?", run.ID, "harness.replan_exhausted").Count(&exhaustedN)
	if exhaustedN < 1 {
		t.Fatal("want harness.replan_exhausted")
	}
}

func TestReplanMaxTurnsHelper(t *testing.T) {
	if replanMaxTurns(rules.Step{Verify: &rules.VerifySpec{MaxReplanTurns: 5}}) != 5 {
		t.Fatal("custom")
	}
	if replanMaxTurns(rules.Step{}) != 3 {
		t.Fatal("default")
	}
}

func TestMaybeProposeReplanFailClosedWithoutEvents(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	now := time.Now().UTC()
	run := store.RunRecord{
		ID: "run_no_ev", SpaceID: "local", TraceID: "tr", Status: "failed",
		ScenarioName: "x", ScenarioVersion: "1", StartedAt: now, CreatedAt: now, UpdatedAt: now,
	}
	if err := db.Create(&run).Error; err != nil {
		t.Fatal(err)
	}
	svc := &Service{db: db}
	step := rules.Step{ID: "qa.verify", Verify: &rules.VerifySpec{OnFail: "replan", MaxReplanTurns: 2}}
	if !svc.maybeProposeReplan(&run, run.ID, step, "no events") {
		t.Fatal("without events service must fail closed")
	}
}

func TestCountReplanProposalsExactStepID(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	now := time.Now().UTC()
	run := store.RunRecord{
		ID: "run_exact", SpaceID: "local", TraceID: "tr", Status: "failed",
		ScenarioName: "x", ScenarioVersion: "1", StartedAt: now, CreatedAt: now, UpdatedAt: now,
	}
	if err := db.Create(&run).Error; err != nil {
		t.Fatal(err)
	}
	ev := events.NewService(db)
	svc := &Service{db: db, events: ev}
	short := rules.Step{ID: "verify", Verify: &rules.VerifySpec{OnFail: "replan", MaxReplanTurns: 3}}
	long := rules.Step{ID: "qa.verify", Verify: &rules.VerifySpec{OnFail: "replan", MaxReplanTurns: 3}}
	if svc.maybeProposeReplan(&run, run.ID, short, "short") {
		t.Fatal("short step should propose")
	}
	nShort, err := svc.countReplanProposals(run.ID, "verify")
	if err != nil || nShort != 1 {
		t.Fatalf("short count=%d err=%v", nShort, err)
	}
	nLong, err := svc.countReplanProposals(run.ID, "qa.verify")
	if err != nil || nLong != 0 {
		t.Fatalf("long must not count short proposals: %d err=%v", nLong, err)
	}
	_ = long
}
