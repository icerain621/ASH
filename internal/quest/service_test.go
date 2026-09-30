package quest

import (
	"testing"
	"time"

	"github.com/ash-repwiki/ash/internal/events"
	"github.com/ash-repwiki/ash/internal/goal"
	"github.com/ash-repwiki/ash/internal/runs"
	"github.com/ash-repwiki/ash/internal/store"
)

func TestColumnForRun(t *testing.T) {
	cases := map[string]string{
		runs.StatusWaitingApproval: "waiting_approval",
		runs.StatusFinished:        "finished",
		runs.StatusFailed:          "finished",
		runs.StatusCanceled:        "finished",
		runs.StatusRunning:         "running",
		"unknown":                  "running",
	}
	for status, want := range cases {
		if got := columnForRun(status); got != want {
			t.Fatalf("columnForRun(%q)=%q want %q", status, got, want)
		}
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate("short", 10); got != "short" {
		t.Fatalf("got %q", got)
	}
	got := truncate("abcdefghijXYZ", 5)
	if got != "abcde…" {
		t.Fatalf("got %q", got)
	}
}

func TestBoardPlansAndRuns(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	now := time.Now().UTC()

	plan := store.GoalPlan{
		ID: "plan_board_1", SpaceID: "local", Goal: "ship quest board coverage",
		Status: goal.StatusDraft, ScenarioName: "feature_delivery", ScenarioVersion: "1.0.0",
		InputsJSON: "{}", StepsJSON: "[]", CreatedAt: now, UpdatedAt: now,
	}
	if err := db.Create(&plan).Error; err != nil {
		t.Fatal(err)
	}
	rejected := store.GoalPlan{
		ID: "plan_board_2", SpaceID: "local", Goal: "rejected plan",
		Status: goal.StatusRejected, ScenarioName: "feature_delivery", ScenarioVersion: "1.0.0",
		InputsJSON: "{}", StepsJSON: "[]", CreatedAt: now, UpdatedAt: now,
	}
	if err := db.Create(&rejected).Error; err != nil {
		t.Fatal(err)
	}

	runID := "run_quest_board_1"
	if err := db.Create(&store.RunRecord{
		ID: runID, TraceID: "tr_quest", ScenarioName: "feature_delivery", ScenarioVersion: "1.0.0",
		PolicyProfile: "default", Status: runs.StatusRunning, SpaceID: "local", ActorRole: "maintainer",
		StartedAt: now, CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	runsSvc := runs.NewService(db, events.NewService(db), nil, nil)
	svc := NewService(db, runsSvc)

	board, err := svc.Board("local", 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(board.Columns["plans"]) == 0 {
		t.Fatal("want draft plan in plans column")
	}
	if len(board.Columns["finished"]) == 0 {
		t.Fatal("want rejected plan in finished")
	}
	foundRun := false
	for _, item := range board.Columns["running"] {
		if item.RunID == runID {
			foundRun = true
		}
	}
	if !foundRun {
		t.Fatalf("run %s missing from running column=%+v", runID, board.Columns)
	}

	limited, err := svc.WithContext(t.Context()).Board("local", 0)
	if err != nil || limited == nil {
		t.Fatalf("default limit board: %v", err)
	}
}
