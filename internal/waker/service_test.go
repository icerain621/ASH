package waker

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/ash-repwiki/ash/internal/runs"
	"github.com/ash-repwiki/ash/internal/store"
)

func TestQueueAndSweepStaleRuns(t *testing.T) {
	t.Setenv("ASH_WAKER_RUN_TTL", "1h")
	db, err := store.Open(filepath.Join(t.TempDir(), "ash.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	now := time.Now().UTC()
	stale := store.RunRecord{
		ID: "run_stale", TraceID: "tr_stale",
		ScenarioName: "feature_delivery", ScenarioVersion: "1.0.0",
		PolicyProfile: "default", Status: runs.StatusRunning, SpaceID: "local",
		RepoRoot: ".", StartedAt: now.Add(-3 * time.Hour),
		CreatedAt: now.Add(-3 * time.Hour), UpdatedAt: now.Add(-3 * time.Hour),
	}
	fresh := store.RunRecord{
		ID: "run_fresh", TraceID: "tr_fresh",
		ScenarioName: "feature_delivery", ScenarioVersion: "1.0.0",
		PolicyProfile: "default", Status: runs.StatusRunning, SpaceID: "local",
		RepoRoot: ".", StartedAt: now, CreatedAt: now, UpdatedAt: now,
	}
	if err := db.Create(&stale).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&fresh).Error; err != nil {
		t.Fatal(err)
	}

	svc := NewService(db)
	q, err := svc.Queue("local", "1h", 20)
	if err != nil {
		t.Fatal(err)
	}
	if q.Count != 1 || q.Items[0].RunID != "run_stale" {
		t.Fatalf("queue=%+v", q)
	}
	if q.Items[0].Kind != KindStaleRun {
		t.Fatalf("kind=%q", q.Items[0].Kind)
	}

	dry := true
	sw, err := svc.Sweep(SweepRequest{SpaceID: "local", DryRun: &dry, MaxAge: "1h"})
	if err != nil || !sw.OK || sw.Matched != 1 || sw.Flagged != 0 {
		t.Fatalf("sweep dry=%+v err=%v", sw, err)
	}
	live := false
	sw2, err := svc.Sweep(SweepRequest{SpaceID: "local", DryRun: &live, MaxAge: "1h"})
	if err != nil || sw2.Flagged != 1 {
		t.Fatalf("sweep live=%+v err=%v", sw2, err)
	}
}

func TestCancelRequiresSafetyGates(t *testing.T) {
	t.Setenv("ASH_WAKER_RUN_TTL", "1h")
	t.Setenv("ASH_WAKER_ALLOW_CANCEL", "")
	db, err := store.Open(filepath.Join(t.TempDir(), "ash.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	now := time.Now().UTC()
	if err := db.Create(&store.RunRecord{
		ID: "run_cancel", TraceID: "tr_c",
		ScenarioName: "feature_delivery", ScenarioVersion: "1.0.0",
		PolicyProfile: "default", Status: runs.StatusRunning, SpaceID: "local",
		RepoRoot: ".", StartedAt: now.Add(-3 * time.Hour),
		CreatedAt: now.Add(-3 * time.Hour), UpdatedAt: now.Add(-3 * time.Hour),
	}).Error; err != nil {
		t.Fatal(err)
	}
	svc := NewService(db)
	dry := false
	_, err = svc.Sweep(SweepRequest{
		SpaceID: "local", DryRun: &dry, MaxAge: "1h",
		Action: "cancel", Confirm: CancelConfirmPhrase,
	})
	if err == nil || !errors.Is(err, ErrCancelDenied) {
		t.Fatalf("want ErrCancelDenied got %v", err)
	}

	t.Setenv("ASH_WAKER_ALLOW_CANCEL", "1")
	_, err = svc.Sweep(SweepRequest{
		SpaceID: "local", DryRun: &dry, MaxAge: "1h",
		Action: "cancel", Confirm: "WRONG",
	})
	if err == nil || !errors.Is(err, ErrCancelDenied) {
		t.Fatalf("want confirm deny got %v", err)
	}

	sw, err := svc.Sweep(SweepRequest{
		SpaceID: "local", DryRun: &dry, MaxAge: "1h",
		Action: "cancel", Confirm: CancelConfirmPhrase,
	})
	if err != nil || sw.Canceled != 1 {
		t.Fatalf("cancel=%+v err=%v", sw, err)
	}
	var rec store.RunRecord
	if err := db.First(&rec, "id = ?", "run_cancel").Error; err != nil {
		t.Fatal(err)
	}
	if rec.Status != runs.StatusCanceled {
		t.Fatalf("status=%s", rec.Status)
	}
}

func TestParseInterval(t *testing.T) {
	if _, ok := ParseInterval("off"); ok {
		t.Fatal("off")
	}
	d, ok := ParseInterval("5m")
	if !ok || d != 5*time.Minute {
		t.Fatalf("%v %v", d, ok)
	}
}
