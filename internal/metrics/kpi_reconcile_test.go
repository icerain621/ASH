package metrics

import (
	"strings"
	"testing"
	"time"

	"github.com/ash-repwiki/ash/internal/observability/derive"
	"github.com/ash-repwiki/ash/internal/store"
)

// TestKPIOverviewMatchesDeriveReplay reconciles KPI-01 counts with derive replay (doc/14-kpi §9).
func TestKPIOverviewMatchesDeriveReplay(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	svc := NewService(db)
	now := time.Now().UTC()
	done := now.Add(2 * time.Second)

	runs := []struct {
		id, status, endType string
	}{
		{"run_ok_a", "finished", "run.finished"},
		{"run_ok_b", "finished", "run.finished"},
		{"run_fail", "failed", "run.failed"},
	}
	for i, seed := range runs {
		fin := done.Add(time.Duration(i) * time.Millisecond)
		row := store.RunRecord{
			ID: seed.id, TraceID: "tr_" + seed.id, SpaceID: "local",
			ScenarioName: "feature_delivery", ScenarioVersion: "1.0.0",
			PolicyProfile: "default", Status: seed.status,
			StartedAt: now, CreatedAt: now, UpdatedAt: now,
		}
		if seed.status == "finished" || seed.status == "failed" {
			row.FinishedAt = &fin
		}
		if err := db.Create(&row).Error; err != nil {
			t.Fatal(err)
		}
		events := []store.RunEvent{
			{
				ID: "ev_start_" + seed.id, RunID: seed.id, Seq: 1, TS: now.UnixMilli(),
				Type: "run.started", Severity: "info",
				PayloadJSON: `{"scenario":{"name":"feature_delivery","scenarioVersion":"1.0.0"},"policyProfile":"default","inputsDigest":"d"}`,
				CreatedAt: now,
			},
			{
				ID: "ev_end_" + seed.id, RunID: seed.id, Seq: 2, TS: fin.UnixMilli(),
				Type: seed.endType, Severity: "info",
				PayloadJSON: `{"ok":` + boolJSON(seed.status == "finished") + `,"durationMs":2000}`,
				CreatedAt: fin,
			},
		}
		for _, ev := range events {
			if err := db.Create(&ev).Error; err != nil {
				t.Fatal(err)
			}
		}
	}

	req := OverviewRequest{SpaceID: "local", From: now.Add(-time.Hour), To: now.Add(time.Hour)}
	overview, err := svc.Overview(req)
	if err != nil {
		t.Fatal(err)
	}
	kpi01 := card(overview, "KPI-01")
	if kpi01.Denominator != 3 || kpi01.Numerator != 2 {
		t.Fatalf("KPI-01=%+v want numerator=2 denominator=3", kpi01)
	}

	events, err := derive.LoadFromDB(db.DB, derive.LoadOptions{SpaceID: "local"})
	if err != nil {
		t.Fatal(err)
	}
	if err := derive.ValidateReplayParity(events); err != nil {
		t.Fatal(err)
	}
	snap := derive.Replay(events)
	var startedReplay, finishedReplay, failedReplay float64
	for key, val := range snap.Counters {
		if !strings.HasPrefix(key, "ash_run_total") {
			continue
		}
		switch {
		case strings.Contains(key, `status="started"`):
			startedReplay += val
		case strings.Contains(key, `status="finished"`):
			finishedReplay += val
		case strings.Contains(key, `status="failed"`):
			failedReplay += val
		}
	}
	if startedReplay != 3 {
		t.Fatalf("replay started=%g want 3", startedReplay)
	}
	if finishedReplay != 2 {
		t.Fatalf("replay finished=%g want 2 (KPI-01 numerator)", finishedReplay)
	}
	if failedReplay != 1 {
		t.Fatalf("replay failed=%g want 1", failedReplay)
	}
	if int64(finishedReplay) != kpi01.Numerator {
		t.Fatalf("KPI-01 numerator=%d replay finished=%g", kpi01.Numerator, finishedReplay)
	}
}

func TestKPIOverviewSummaryCatalog(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	svc := NewService(db)
	now := time.Now().UTC()
	if err := db.Create(&store.RunRecord{
		ID: "run_cat", TraceID: "tr_cat", SpaceID: "local",
		ScenarioName: "feature_delivery", ScenarioVersion: "1.0.0",
		Status: "finished", StartedAt: now, CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	overview, err := svc.Overview(OverviewRequest{
		SpaceID: "local", From: now.Add(-time.Hour), To: now.Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"KPI-01", "KPI-02", "KPI-03", "KPI-04", "KPI-05", "KPI-06", "KPI-07", "KPI-08", "KPI-09", "KPI-10"}
	seen := map[string]bool{}
	for _, c := range overview.Summary {
		seen[c.ID] = true
	}
	for _, id := range want {
		if !seen[id] {
			t.Fatalf("overview missing %s; summary=%d cards", id, len(overview.Summary))
		}
	}
}

func boolJSON(ok bool) string {
	if ok {
		return "true"
	}
	return "false"
}
