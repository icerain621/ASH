package events_test

import (
	"testing"
	"time"

	"github.com/ash-repwiki/ash/internal/events"
	"github.com/ash-repwiki/ash/internal/store"
)

func TestDeriveModelVisible_filtersByVisibility(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	ev := events.NewService(db)
	runID := "run_mv_1"

	if _, err := ev.Append(runID, "tr", "session.turn", "info", map[string]any{"prompt": "hi"}); err != nil {
		t.Fatal(err)
	}
	if _, err := ev.Append(runID, "tr", "gate.waiting_approval", "warn", map[string]any{"reason": "x"}); err != nil {
		t.Fatal(err)
	}
	if _, err := ev.Append(runID, "tr", "metric.kpi", "info", map[string]any{"n": 1}); err != nil {
		t.Fatal(err)
	}
	if _, err := ev.Append(runID, "tr", "session.turn", "info", map[string]any{"prompt": "secret"}, events.WithVisibility(events.VisibilityAudit)); err != nil {
		t.Fatal(err)
	}

	got, err := ev.DeriveModelVisible(runID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("len=%d want 1 model_visible only; got=%+v", len(got), got)
	}
	if got[0].Type != "session.turn" || got[0].Visibility != events.VisibilityModelVisible {
		t.Fatalf("got=%+v", got[0])
	}
}

func TestDeriveModelVisible_legacyEmptyDefaults(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	now := time.Now().UTC()
	row := store.RunEvent{
		ID: "evt_legacy_mv", RunID: "run_mv_legacy", Seq: 1,
		TS: now.UnixMilli(), Type: "run.started", Severity: "info",
		PayloadJSON: `{"status":"running"}`, Visibility: "", CreatedAt: now,
	}
	if err := db.Create(&row).Error; err != nil {
		t.Fatal(err)
	}
	ev := events.NewService(db)
	got, err := ev.DeriveModelVisible("run_mv_legacy")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Visibility != events.VisibilityModelVisible {
		t.Fatalf("got=%+v", got)
	}
}
