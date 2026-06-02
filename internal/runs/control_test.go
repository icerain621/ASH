package runs

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/ash-repwiki/ash/internal/events"
	"github.com/ash-repwiki/ash/internal/rules"
	"github.com/ash-repwiki/ash/internal/store"
	"github.com/ash-repwiki/ash/internal/toolbus"
)

func testRunsService(t *testing.T) (*Service, string) {
	t.Helper()
	dir := t.TempDir()
	db, err := store.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	scenariosDir := filepath.Join("..", "..", "scenarios")
	if _, err := os.Stat(scenariosDir); err != nil {
		scenariosDir = "scenarios"
	}
	loader := rules.NewLoader(scenariosDir)
	if err := loader.LoadDir(); err != nil {
		t.Fatal(err)
	}
	ev := events.NewService(db)
	return NewService(db, ev, loader, toolbus.DefaultBus()), scenariosDir
}

func TestReplayExact(t *testing.T) {
	svc, _ := testRunsService(t)
	createReq := CreateRequest{
		Scenario: ScenarioRef{Name: "feature_delivery", ScenarioVersion: "1.0.0"},
		Inputs: map[string]any{
			"issueOrSpec": "replay test",
			"repoRoot":    ".",
		},
	}
	orig, err := svc.Create(createReq)
	if err != nil {
		t.Fatal(err)
	}

	replayed, err := svc.Replay(orig.RunID, ReplayRequest{Mode: "exact"})
	if err != nil {
		t.Fatal(err)
	}
	if replayed.RunID == orig.RunID {
		t.Fatal("replay should create a new run id")
	}

	meta, err := loadRunMeta(svc.db.RunDir(replayed.RunID))
	if err != nil {
		t.Fatal(err)
	}
	if meta.SourceRunID != orig.RunID {
		t.Fatalf("sourceRunId=%q want %q", meta.SourceRunID, orig.RunID)
	}
	if meta.ReplayMode != "exact" {
		t.Fatalf("replayMode=%q want exact", meta.ReplayMode)
	}

	sum, err := svc.Get(replayed.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if sum.Status != "finished" {
		t.Fatalf("status=%q want finished", sum.Status)
	}
}

func TestResumeFailedRun(t *testing.T) {
	svc, _ := testRunsService(t)
	createReq := CreateRequest{
		Scenario: ScenarioRef{Name: "feature_delivery", ScenarioVersion: "1.0.0"},
		Inputs: map[string]any{
			"issueOrSpec": "resume test",
			"repoRoot":    ".",
		},
	}
	created, err := svc.Create(createReq)
	if err != nil {
		t.Fatal(err)
	}

	var rec store.RunRecord
	if err := svc.db.First(&rec, "id = ?", created.RunID).Error; err != nil {
		t.Fatal(err)
	}
	rec.Status = "failed"
	rec.ErrorCode = "TEST_FAIL"
	rec.ErrorMessage = "simulated failure"
	if err := svc.db.Save(&rec).Error; err != nil {
		t.Fatal(err)
	}

	resp, err := svc.Resume(created.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if resp.RunID != created.RunID {
		t.Fatalf("resume runId=%q want same %q", resp.RunID, created.RunID)
	}
	if resp.Status != "finished" {
		t.Fatalf("status=%q want finished", resp.Status)
	}

	evs, err := svc.events.ListAfter(created.RunID, 0, 500)
	if err != nil {
		t.Fatal(err)
	}
	foundResumed := false
	for _, ev := range evs {
		if ev.Type == "run.resumed" {
			foundResumed = true
			break
		}
	}
	if !foundResumed {
		t.Fatal("expected run.resumed event")
	}
}

func TestResumeNotResumable(t *testing.T) {
	svc, _ := testRunsService(t)
	createReq := CreateRequest{
		Scenario: ScenarioRef{Name: "feature_delivery", ScenarioVersion: "1.0.0"},
		Inputs: map[string]any{
			"issueOrSpec": "not resumable",
			"repoRoot":    ".",
		},
	}
	created, err := svc.Create(createReq)
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.Resume(created.RunID)
	if err == nil {
		t.Fatal("expected error for finished run")
	}
	if !errors.Is(err, ErrRunNotResumable) {
		t.Fatalf("err=%v want ErrRunNotResumable", err)
	}
}
