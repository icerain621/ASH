package events_test

import (
	"testing"
	"time"

	"github.com/ash-repwiki/ash/internal/events"
	"github.com/ash-repwiki/ash/internal/store"
)

func TestDefaultVisibility_byEventType(t *testing.T) {
	cases := []struct {
		eventType string
		want      string
	}{
		{"session.turn", events.VisibilityModelVisible},
		{"run.started", events.VisibilityModelVisible},
		{"gate.approved", events.VisibilityModelVisible},
		{"gate.waiting_approval", events.VisibilityUIOnly},
		{"metric.kpi", events.VisibilityAudit},
		{"score.recorded", events.VisibilityAudit},
		{"audit.access", events.VisibilityAudit},
		{"ui.toast", events.VisibilityUIOnly},
	}
	for _, tc := range cases {
		if got := events.DefaultVisibility(tc.eventType); got != tc.want {
			t.Fatalf("DefaultVisibility(%q)=%q want %q", tc.eventType, got, tc.want)
		}
	}
}

func TestNormalizeVisibility_emptyUsesDefault(t *testing.T) {
	got := events.NormalizeVisibility("session.turn", "")
	if got != events.VisibilityModelVisible {
		t.Fatalf("got %q", got)
	}
	got = events.NormalizeVisibility("metric.x", "  ")
	if got != events.VisibilityAudit {
		t.Fatalf("got %q", got)
	}
	got = events.NormalizeVisibility("session.turn", events.VisibilityUIOnly)
	if got != events.VisibilityUIOnly {
		t.Fatalf("got %q", got)
	}
}

func TestNormalizeVisibility_invalidFallsBack(t *testing.T) {
	got := events.NormalizeVisibility("session.turn", "nope")
	if got != events.VisibilityModelVisible {
		t.Fatalf("got %q want default for type", got)
	}
}

func TestAppend_setsDefaultVisibility(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	ev := events.NewService(db)
	env, err := ev.Append("run_vis_1", "trace_vis_1", "session.turn", "info", map[string]any{
		"prompt": "hi",
	})
	if err != nil {
		t.Fatal(err)
	}
	if env.Visibility != events.VisibilityModelVisible {
		t.Fatalf("envelope visibility=%q", env.Visibility)
	}

	var row store.RunEvent
	if err := db.First(&row, "id = ?", env.ID).Error; err != nil {
		t.Fatal(err)
	}
	if row.Visibility != events.VisibilityModelVisible {
		t.Fatalf("persisted visibility=%q", row.Visibility)
	}
}

func TestAppend_withExplicitVisibility(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	ev := events.NewService(db)
	env, err := ev.Append("run_vis_2", "trace_vis_2", "session.turn", "info", map[string]any{
		"prompt": "secret-ish",
	}, events.WithVisibility(events.VisibilityAudit))
	if err != nil {
		t.Fatal(err)
	}
	if env.Visibility != events.VisibilityAudit {
		t.Fatalf("envelope visibility=%q", env.Visibility)
	}
}

func TestListAfter_fillsLegacyEmptyVisibility(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	now := time.Now().UTC()
	row := store.RunEvent{
		ID: "evt_legacy_vis", RunID: "run_vis_legacy", Seq: 1,
		TS: now.UnixMilli(), Type: "run.started", Severity: "info",
		PayloadJSON: `{"status":"running"}`, Visibility: "", CreatedAt: now,
	}
	if err := db.Create(&row).Error; err != nil {
		t.Fatal(err)
	}
	ev := events.NewService(db)
	items, err := ev.ListAfter("run_vis_legacy", 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("len=%d", len(items))
	}
	if items[0].Visibility != events.VisibilityModelVisible {
		t.Fatalf("visibility=%q want model_visible default", items[0].Visibility)
	}
}
