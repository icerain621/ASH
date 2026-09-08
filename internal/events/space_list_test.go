package events_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/ash-repwiki/ash/internal/events"
	"github.com/ash-repwiki/ash/internal/goal"
	"github.com/ash-repwiki/ash/internal/rules"
	"github.com/ash-repwiki/ash/internal/runs"
	"github.com/ash-repwiki/ash/internal/store"
	"github.com/ash-repwiki/ash/internal/toolbus"
)

func TestListAfterSpace_planCreated(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	loader := rules.NewLoader(filepath.Join("..", "..", "scenarios"))
	if err := loader.LoadDir(); err != nil {
		t.Fatal(err)
	}
	ev := events.NewService(db)
	runsSvc := runs.NewService(db, ev, loader, toolbus.DefaultBus())
	goalSvc := goal.NewService(db, loader, runsSvc, ev)

	before := time.Now().UTC().UnixMilli() - 1
	plan, err := goalSvc.FromGoal(goal.FromGoalRequest{
		Goal: "Add export CSV to reports", SpaceID: "local", CreatedBy: "dx53",
	})
	if err != nil {
		t.Fatal(err)
	}

	got, err := ev.ListAfterSpace("local", before, 50)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, e := range got {
		if e.Type == "plan.created" && e.RunID == plan.ID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("want plan.created for %s in %+v", plan.ID, got)
	}

	// Future cursor should see nothing yet.
	later, err := ev.ListAfterSpace("local", time.Now().UTC().UnixMilli()+1000, 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(later) != 0 {
		t.Fatalf("future cursor got %d events", len(later))
	}
}
