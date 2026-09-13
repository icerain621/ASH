package session_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/ash-repwiki/ash/internal/events"
	"github.com/ash-repwiki/ash/internal/session"
	"github.com/ash-repwiki/ash/internal/store"
)

type stubRunControl struct {
	approveCalls int
	cancelCalls  int
	approveErr   error
	cancelErr    error
	lastActor    string
	lastReason   string
	lastRunID    string
}

func (s *stubRunControl) ApproveRun(runID, actorID, reason string) error {
	s.approveCalls++
	s.lastRunID = runID
	s.lastActor = actorID
	s.lastReason = reason
	return s.approveErr
}

func (s *stubRunControl) CancelRun(runID string) error {
	s.cancelCalls++
	s.lastRunID = runID
	return s.cancelErr
}

func seedRun(t *testing.T, db *store.DB, id, status string) {
	t.Helper()
	now := time.Now().UTC()
	run := store.RunRecord{
		ID: id, TraceID: "trace_" + id,
		ScenarioName: "feature_delivery", ScenarioVersion: "1.0.0",
		PolicyProfile: "default", Status: status, SpaceID: "local",
		RepoRoot: ".", StartedAt: now, CreatedAt: now, UpdatedAt: now,
	}
	if err := db.Create(&run).Error; err != nil {
		t.Fatal(err)
	}
}

func TestCreate_ensuresMainThread(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	ev := events.NewService(db)
	svc := session.NewService(db, nil, ev)
	seedRun(t, db, "run_th_1", "running")
	view, err := svc.Create(session.CreateRequest{RunID: "run_th_1", SpaceID: "local", CreatedBy: "test"})
	if err != nil {
		t.Fatal(err)
	}
	tid, _ := view.Meta["threadId"].(string)
	if tid == "" || view.Meta["threadKind"] != "main" {
		t.Fatalf("meta=%v want main thread", view.Meta)
	}
	view2, err := svc.Get(view.ID)
	if err != nil {
		t.Fatal(err)
	}
	if view2.Meta["threadId"] != tid {
		t.Fatalf("persisted threadId=%v want %s", view2.Meta["threadId"], tid)
	}
}

func TestIntent_promptApproveCancel(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	ev := events.NewService(db)
	rc := &stubRunControl{}
	svc := session.NewService(db, nil, ev).WithRunControl(rc)
	seedRun(t, db, "run_intent_1", "waiting_approval")
	view, err := svc.Create(session.CreateRequest{RunID: "run_intent_1", SpaceID: "local", CreatedBy: "actor1"})
	if err != nil {
		t.Fatal(err)
	}

	out, err := svc.Intent(view.ID, session.IntentRequest{Action: "prompt", Prompt: "next step", ActorID: "actor1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Turns) != 1 {
		t.Fatalf("turns=%d", len(out.Turns))
	}

	out, err = svc.Intent(view.ID, session.IntentRequest{Action: "approve", Reason: "looks good", ActorID: "actor1"})
	if err != nil {
		t.Fatal(err)
	}
	if rc.approveCalls != 1 || rc.lastRunID != "run_intent_1" || rc.lastReason != "looks good" {
		t.Fatalf("approve stub=%+v", rc)
	}
	if out.ID != view.ID {
		t.Fatalf("out=%+v", out)
	}

	_, err = svc.Intent(view.ID, session.IntentRequest{Action: "cancel", ActorID: "actor1"})
	if err != nil {
		t.Fatal(err)
	}
	if rc.cancelCalls != 1 {
		t.Fatalf("cancelCalls=%d", rc.cancelCalls)
	}
}

func TestIntent_approveFailClosedWithoutGate(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	ev := events.NewService(db)
	rc := &stubRunControl{approveErr: errors.New(`run is not approvable: status is "running"`)}
	svc := session.NewService(db, nil, ev).WithRunControl(rc)
	seedRun(t, db, "run_intent_2", "running")
	view, err := svc.Create(session.CreateRequest{RunID: "run_intent_2", SpaceID: "local", CreatedBy: "actor1"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.Intent(view.ID, session.IntentRequest{Action: "approve", ActorID: "actor1", Reason: "x"})
	if err == nil {
		t.Fatal("expected approve to fail closed")
	}
	if !errors.Is(err, session.ErrIntentRejected) && !strings.Contains(err.Error(), "approvable") {
		t.Fatalf("err=%v want ErrIntentRejected or approvable message", err)
	}
}
