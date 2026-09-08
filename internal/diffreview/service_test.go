package diffreview

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ash-repwiki/ash/internal/events"
	"github.com/ash-repwiki/ash/internal/runs"
	"github.com/ash-repwiki/ash/internal/rules"
	"github.com/ash-repwiki/ash/internal/store"
	"github.com/ash-repwiki/ash/internal/toolbus"
)

func testRunService(t *testing.T, db *store.DB) *runs.Service {
	t.Helper()
	ev := events.NewService(db)
	loader := rules.NewLoader("scenarios")
	return runs.NewService(db, ev, loader, toolbus.DefaultBus())
}

func TestRejectFileRecordsComment(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	runSvc := testRunService(t, db)
	svc := NewService(db, runSvc)

	now := time.Now().UTC()
	runID := "run_diff_file"
	if err := db.Create(&store.RunRecord{
		ID: runID, TraceID: "trc_df", ScenarioName: "feature_delivery", ScenarioVersion: "1.0.0",
		PolicyProfile: "default", Status: "finished", SpaceID: "local",
		StartedAt: now, CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	artDir := filepath.Join(db.RunDir(runID), "artifacts")
	if err := os.MkdirAll(artDir, 0o755); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(artDir, "diff.patch"), []byte("diff --git a/a.go b/a.go\n"), 0o644)

	resp, err := svc.Reject(runID, RejectRequest{Scope: RejectScopeFile, FilePath: "a.go", Reason: "needs tests", ActorID: "tester"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Canceled {
		t.Fatal("file reject must not cancel")
	}
	if resp.Comment.Side != RejectSide || resp.Comment.FilePath != "a.go" {
		t.Fatalf("comment=%+v", resp.Comment)
	}
	paths, err := svc.RejectedPaths(runID)
	if err != nil || len(paths) != 1 || paths[0] != "a.go" {
		t.Fatalf("paths=%v err=%v", paths, err)
	}
}

func TestRejectAllCancelsRun(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	runSvc := testRunService(t, db)
	svc := NewService(db, runSvc)

	now := time.Now().UTC()
	runID := "run_diff_all"
	if err := db.Create(&store.RunRecord{
		ID: runID, TraceID: "trc_da", ScenarioName: "hotfix", ScenarioVersion: "1.0.0",
		PolicyProfile: "default", Status: runs.StatusWaitingApproval, SpaceID: "local",
		StartedAt: now, CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	resp, err := svc.Reject(runID, RejectRequest{Scope: RejectScopeAll, Reason: "wrong approach", ActorID: "tester"})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Canceled || resp.Status != runs.StatusCanceled {
		t.Fatalf("resp=%+v", resp)
	}
	paths, err := svc.RejectedPaths(runID)
	if err != nil || len(paths) != 1 || paths[0] != RejectAllPath {
		t.Fatalf("paths=%v err=%v", paths, err)
	}
}
