package goal_test

import (
	"path/filepath"
	"testing"

	"github.com/ash-repwiki/ash/internal/events"
	"github.com/ash-repwiki/ash/internal/goal"
	"github.com/ash-repwiki/ash/internal/rules"
	"github.com/ash-repwiki/ash/internal/runs"
	"github.com/ash-repwiki/ash/internal/store"
	"github.com/ash-repwiki/ash/internal/toolbus"
)

func TestFromGoalApproveStartsRun(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	loader := rules.NewLoader(filepath.Join("..", "..", "scenarios"))
	if err := loader.LoadDir(); err != nil {
		t.Fatal(err)
	}
	ev := events.NewService(db)
	runsSvc := runs.NewService(db, ev, loader, toolbus.DefaultBus())
	svc := goal.NewService(db, loader, runsSvc, ev)

	plan, err := svc.FromGoal(goal.FromGoalRequest{
		Goal: "Add export CSV to reports", SpaceID: "local", CreatedBy: "test",
	})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Status != goal.StatusDraft {
		t.Fatalf("status=%s", plan.Status)
	}
	if plan.ScenarioName != "feature_delivery" {
		t.Fatalf("scenario=%s", plan.ScenarioName)
	}
	if plan.RunID != "" {
		t.Fatal("draft must not start run")
	}

	approved, err := svc.Approve(plan.ID, "tester", "ok", "maintainer")
	if err != nil && approved == nil {
		t.Fatal(err)
	}
	if approved.RunID == "" {
		t.Fatal("expected runId after approve")
	}
	if approved.Status != goal.StatusStarted {
		t.Fatalf("status=%s", approved.Status)
	}
}

func TestFromGoalAutoApprove(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	loader := rules.NewLoader(filepath.Join("..", "..", "scenarios"))
	if err := loader.LoadDir(); err != nil {
		t.Fatal(err)
	}
	ev := events.NewService(db)
	runsSvc := runs.NewService(db, ev, loader, toolbus.DefaultBus())
	svc := goal.NewService(db, loader, runsSvc, ev)

	plan, err := svc.FromGoal(goal.FromGoalRequest{
		Goal: "hotfix prod outage checkout", SpaceID: "local", AutoApprove: true, CreatedBy: "cli",
	})
	if err != nil && plan == nil {
		t.Fatal(err)
	}
	if plan.ScenarioName != "hotfix" {
		t.Fatalf("scenario=%s", plan.ScenarioName)
	}
	if plan.RunID == "" {
		t.Fatal("autoApprove should start run")
	}
}
