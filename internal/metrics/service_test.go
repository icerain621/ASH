package metrics

import (
	"testing"
	"time"

	"github.com/ash-repwiki/ash/internal/store"
)

func TestOverviewAggregatesKPIInputs(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	svc := NewService(db)
	now := time.Now().UTC()
	done := now.Add(time.Second)
	if err := db.Create(&store.RunRecord{
		ID: "run_ok", TraceID: "trace_ok", ScenarioName: "feature_delivery", ScenarioVersion: "1.0.0",
		PolicyProfile: "default", Status: "finished", SpaceID: "local",
		StartedAt: now.Add(-time.Hour), FinishedAt: &done, CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&store.Feedback{
		ID: "fb_low", SpaceID: "local", TargetType: "run", TargetID: "run_ok", Rating: 1, CreatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&store.Feedback{
		ID: "fb_suggestion", SpaceID: "local", TargetType: "suggestion", TargetID: "s1", Rating: 5, Comment: "accepted", CreatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&store.RunEvent{
		ID: "ev_memory", RunID: "run_ok", Seq: 1, TS: now.UnixMilli(), Type: "memory.injected", Severity: "info",
		PayloadJSON: "{}", CreatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&store.AuditLog{
		ID: "aud_memory", SpaceID: "local", RunID: "run_ok", EventType: "memory.hit_used", PayloadJSON: "{}", CreatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&store.CIRun{
		ID: "ci_run", SpaceID: "local", ConnectionID: "repo_conn", ProviderRunID: "100",
		Status: "completed", Conclusion: "success", Attempt: 1, CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&store.CIDiagnosis{
		ID: "ci_diag", SpaceID: "local", ConnectionID: "repo_conn", Status: "diagnosed",
		RootCause: "test_failure", FixSuggestionsJSON: "[]", EvidenceRefsJSON: "[]", Adopted: true,
		CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	overview, err := svc.Overview(OverviewRequest{SpaceID: "local", From: now.Add(-2 * time.Hour), To: now.Add(2 * time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	if card(overview, "KPI-01").Value != 1 {
		t.Fatalf("KPI-01=%+v want 1", card(overview, "KPI-01"))
	}
	if card(overview, "KPI-04").Value != 1 || card(overview, "KPI-05").Value != 1 {
		t.Fatalf("ci cards=%+v/%+v want 1", card(overview, "KPI-04"), card(overview, "KPI-05"))
	}
	if card(overview, "KPI-08").Status != "unavailable" {
		t.Fatalf("KPI-08=%+v want unavailable without stream audits", card(overview, "KPI-08"))
	}
}

func TestOverviewSSEStabilityFromStreamAudits(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	svc := NewService(db)
	now := time.Now().UTC()
	for _, row := range []store.AuditLog{
		{ID: "aud_sse_open", SpaceID: "local", RunID: "run_sse", EventType: "stream.session_opened", PayloadJSON: `{"runId":"run_sse"}`, CreatedAt: now},
		{ID: "aud_sse_close", SpaceID: "local", RunID: "run_sse", EventType: "stream.session_closed", PayloadJSON: `{"runId":"run_sse","reason":"client_disconnect"}`, CreatedAt: now},
		{ID: "aud_sse_fail", SpaceID: "local", RunID: "run_sse_bad", EventType: "stream.session_failed", PayloadJSON: `{"runId":"run_sse_bad","reason":"event_poll_failed"}`, CreatedAt: now},
	} {
		if err := db.Create(&row).Error; err != nil {
			t.Fatal(err)
		}
	}
	overview, err := svc.Overview(OverviewRequest{SpaceID: "local", From: now.Add(-time.Hour), To: now.Add(time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	kpi := card(overview, "KPI-08")
	// 1 closed + 1 failed → success rate 0.5 (failed sessions count in denominator)
	if kpi.Status != "ok" || kpi.Numerator != 1 || kpi.Denominator != 2 || kpi.Value != 0.5 {
		t.Fatalf("KPI-08=%+v want ok numerator=1 denominator=2 value=0.5", kpi)
	}
}

func TestOverviewSSEStabilityInFlightOnlyIsEmpty(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	svc := NewService(db)
	now := time.Now().UTC()
	if err := db.Create(&store.AuditLog{
		ID: "aud_sse_open_only", SpaceID: "local", RunID: "run_live",
		EventType: "stream.session_opened", PayloadJSON: `{}`, CreatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	overview, err := svc.Overview(OverviewRequest{SpaceID: "local", From: now.Add(-time.Hour), To: now.Add(time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	kpi := card(overview, "KPI-08")
	if kpi.Status != "empty" {
		t.Fatalf("KPI-08=%+v want empty while only in-flight opens", kpi)
	}
}

func TestOverviewScopesRunEventsAndStepsBySpace(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	svc := NewService(db)
	now := time.Now().UTC()
	localStarted := now.Add(5 * time.Minute)
	otherStarted := now.Add(20 * time.Minute)

	seeds := []struct {
		runID, spaceID, eventID, stepID string
		stepStarted                     time.Time
	}{
		{"run_local_mem", "local", "ev_local_mem", "step_local", localStarted},
		{"run_other_mem", "space_other", "ev_other_mem", "step_other", otherStarted},
	}
	for _, seed := range seeds {
		if err := db.Create(&store.RunRecord{
			ID: seed.runID, TraceID: "tr_" + seed.runID, ScenarioName: "feature_delivery", ScenarioVersion: "1.0.0",
			PolicyProfile: "default", Status: "finished", SpaceID: seed.spaceID,
			StartedAt: now, CreatedAt: now, UpdatedAt: now,
		}).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Create(&store.RunEvent{
			ID: seed.eventID, RunID: seed.runID, Seq: 1, TS: now.UnixMilli(), Type: "memory.injected",
			Severity: "info", PayloadJSON: "{}", CreatedAt: now,
		}).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Create(&store.RunStep{
			ID: seed.stepID, RunID: seed.runID, StepID: "s1", StepOrder: 1, Kind: "tool", Status: "finished",
			CreatedAt: now, StartedAt: &seed.stepStarted,
		}).Error; err != nil {
			t.Fatal(err)
		}
	}

	req := OverviewRequest{SpaceID: "local", From: now.Add(-time.Hour), To: now.Add(2 * time.Hour)}
	overview, err := svc.Overview(req)
	if err != nil {
		t.Fatal(err)
	}
	kpi07 := card(overview, "KPI-07")
	if kpi07.Denominator != 1 {
		t.Fatalf("KPI-07 denominator=%d want 1 (scoped memory.injected)", kpi07.Denominator)
	}
	kpi10 := card(overview, "KPI-10")
	if kpi10.Denominator != 1 {
		t.Fatalf("KPI-10 denominator=%d want 1 (scoped run_steps)", kpi10.Denominator)
	}
	wantWaitMs := int64(5 * time.Minute / time.Millisecond)
	if kpi10.Numerator != wantWaitMs {
		t.Fatalf("KPI-10 numerator=%d want %d (exclude foreign space queue wait)", kpi10.Numerator, wantWaitMs)
	}
}

func TestOverviewScenarioStabilityR02(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	svc := NewService(db)
	now := time.Now().UTC()
	done := now.Add(time.Minute)
	seed := []store.RunRecord{
		{ID: "run_a1", TraceID: "tr_a1", SpaceID: "local", ScenarioName: "feature_delivery", ScenarioVersion: "1.0.0", Status: "finished", StartedAt: now.Add(-3 * time.Hour), FinishedAt: &done, CreatedAt: now, UpdatedAt: now},
		{ID: "run_a2", TraceID: "tr_a2", SpaceID: "local", ScenarioName: "feature_delivery", ScenarioVersion: "1.0.0", Status: "failed", StartedAt: now.Add(-2 * time.Hour), FinishedAt: &done, CreatedAt: now, UpdatedAt: now},
		{ID: "run_a3", TraceID: "tr_a3", SpaceID: "local", ScenarioName: "feature_delivery", ScenarioVersion: "1.0.0", Status: "finished", StartedAt: now.Add(-time.Hour), FinishedAt: &done, CreatedAt: now, UpdatedAt: now},
		{ID: "run_b1", TraceID: "tr_b1", SpaceID: "local", ScenarioName: "hotfix", ScenarioVersion: "1.1.0", Status: "finished", StartedAt: now.Add(-30 * time.Minute), FinishedAt: &done, CreatedAt: now, UpdatedAt: now},
		{ID: "run_b2", TraceID: "tr_b2", SpaceID: "local", ScenarioName: "hotfix", ScenarioVersion: "1.1.0", Status: "finished", StartedAt: now.Add(-20 * time.Minute), FinishedAt: &done, CreatedAt: now, UpdatedAt: now},
	}
	for i := range seed {
		if err := db.Create(&seed[i]).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Create(&store.Feedback{
		ID: "fb_r02", SpaceID: "local", TargetType: "run", TargetID: "run_a2", Rating: 1, CreatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	overview, err := svc.Overview(OverviewRequest{SpaceID: "local", From: now.Add(-4 * time.Hour), To: now.Add(time.Hour)})
	if err != nil {
		t.Fatal(err)
	}

	// feature_delivery 2/3 < 0.85 → unstable; hotfix 2/2 ≥ 0.85 → stable → KPI-11 = 1/2
	kpi11 := card(overview, "KPI-11")
	if kpi11.Status != "ok" || kpi11.Numerator != 1 || kpi11.Denominator != 2 || kpi11.Value != 0.5 {
		t.Fatalf("KPI-11=%+v want numerator=1 denominator=2 value=0.5", kpi11)
	}

	var stab *MetricBreakdown
	for i := range overview.Breakdowns {
		if overview.Breakdowns[i].ID == "scenarioStability" {
			stab = &overview.Breakdowns[i]
			break
		}
	}
	if stab == nil {
		t.Fatal("missing scenarioStability breakdown")
	}
	byKey := map[string]BreakdownItem{}
	for _, item := range stab.Items {
		byKey[item.Key] = item
	}
	if byKey["feature_delivery@1.0.0"].Value != ratio(2, 3) {
		t.Fatalf("feature_delivery rate=%v want %v", byKey["feature_delivery@1.0.0"].Value, ratio(2, 3))
	}
	if byKey["feature_delivery@1.0.0:n"].Value != 3 {
		t.Fatalf("feature_delivery samples=%v want 3", byKey["feature_delivery@1.0.0:n"].Value)
	}
	if byKey["feature_delivery@1.0.0:low"].Value != 1 {
		t.Fatalf("feature_delivery low=%v want 1", byKey["feature_delivery@1.0.0:low"].Value)
	}
	if byKey["hotfix@1.1.0"].Value != 1 {
		t.Fatalf("hotfix rate=%v want 1", byKey["hotfix@1.1.0"].Value)
	}
}

func TestOverviewSandboxCoverageKPI19(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	svc := NewService(db)
	now := time.Now().UTC()
	if err := db.Create(&store.RunRecord{
		ID: "run_kpi19", TraceID: "tr_kpi19", SpaceID: "local",
		ScenarioName: "hotfix", ScenarioVersion: "1.1.0",
		Status: "finished", StartedAt: now, CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	events := []store.RunEvent{
		{ID: "ev1", RunID: "run_kpi19", Seq: 1, TS: now.UnixMilli(), Type: "harness.tool.routed", Severity: "info",
			PayloadJSON: `{"risk":"danger","sandboxMode":"isolated","tool":"runtime.command"}`, CreatedAt: now},
		{ID: "ev2", RunID: "run_kpi19", Seq: 2, TS: now.UnixMilli(), Type: "harness.tool.routed", Severity: "info",
			PayloadJSON: `{"risk":"danger","sandboxMode":"workspace-write","tool":"bash"}`, CreatedAt: now},
		{ID: "ev3", RunID: "run_kpi19", Seq: 3, TS: now.UnixMilli(), Type: "harness.tool.routed", Severity: "info",
			PayloadJSON: `{"risk":"safe","sandboxMode":"off","tool":"read"}`, CreatedAt: now},
	}
	for i := range events {
		if err := db.Create(&events[i]).Error; err != nil {
			t.Fatal(err)
		}
	}
	overview, err := svc.Overview(OverviewRequest{SpaceID: "local", From: now.Add(-time.Hour), To: now.Add(time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	kpi := card(overview, "KPI-19")
	if kpi.Numerator != 1 || kpi.Denominator != 2 || kpi.Value != 0.5 {
		t.Fatalf("KPI-19=%+v want 1/2", kpi)
	}
}

func card(overview Overview, id string) MetricCard {
	for _, item := range overview.Summary {
		if item.ID == id {
			return item
		}
	}
	return MetricCard{}
}
