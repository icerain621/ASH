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
		t.Fatalf("KPI-08=%+v want unavailable", card(overview, "KPI-08"))
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
