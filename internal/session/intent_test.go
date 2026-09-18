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
	lastScope    string
	lastTool     string
}

func (s *stubRunControl) ApproveRun(runID string, req session.GateApproveRequest) error {
	s.approveCalls++
	s.lastRunID = runID
	s.lastActor = req.ActorID
	s.lastReason = req.Reason
	s.lastScope = req.Scope
	s.lastTool = req.Tool
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
	if rc.lastScope != session.ApproveScopeOnce {
		t.Fatalf("default scope=%q want once", rc.lastScope)
	}
	if out.ID != view.ID {
		t.Fatalf("out=%+v", out)
	}

	_, err = svc.Intent(view.ID, session.IntentRequest{
		Action: "allow_session", Reason: "trust bash", ActorID: "actor1", Tool: "bash",
	})
	if err != nil {
		t.Fatal(err)
	}
	if rc.approveCalls != 2 || rc.lastScope != session.ApproveScopeSession || rc.lastTool != "bash" {
		t.Fatalf("allow_session stub=%+v", rc)
	}
	// Session meta allow-list is written by runs.Approve via SessionLinker (evidence tool),
	// not from the client-supplied intent tool name.

	_, err = svc.Intent(view.ID, session.IntentRequest{Action: "cancel", ActorID: "actor1"})
	if err != nil {
		t.Fatal(err)
	}
	if rc.cancelCalls != 1 {
		t.Fatalf("cancelCalls=%d", rc.cancelCalls)
	}

	_, err = svc.Intent(view.ID, session.IntentRequest{Action: "stop", ActorID: "actor1"})
	if err != nil {
		t.Fatal(err)
	}
	if rc.cancelCalls != 2 {
		t.Fatalf("stop alias cancelCalls=%d want 2", rc.cancelCalls)
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

func TestIntent_steerCancelsActiveRunAndPrompts(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	ev := events.NewService(db)
	rc := &stubRunControl{}
	svc := session.NewService(db, nil, ev).WithRunControl(rc)
	seedRun(t, db, "run_steer_1", "running")
	view, err := svc.Create(session.CreateRequest{RunID: "run_steer_1", SpaceID: "local", CreatedBy: "actor1"})
	if err != nil {
		t.Fatal(err)
	}

	out, err := svc.Intent(view.ID, session.IntentRequest{
		Action: "steer", Prompt: "change direction", ActorID: "actor1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if rc.cancelCalls != 1 || rc.lastRunID != "run_steer_1" {
		t.Fatalf("cancel stub=%+v", rc)
	}
	if len(out.Turns) != 1 || out.Turns[0].Prompt != "change direction" {
		t.Fatalf("turns=%+v", out.Turns)
	}

	listed, err := ev.ListAfter("run_steer_1", 0, 50)
	if err != nil {
		t.Fatal(err)
	}
	sawSteer := false
	for _, item := range listed {
		if item.Type == "session.steer" {
			sawSteer = true
			if !strings.Contains(string(item.Payload), "change direction") {
				t.Fatalf("steer payload=%s", item.Payload)
			}
			if item.Visibility != events.VisibilityUIOnly {
				t.Fatalf("visibility=%q", item.Visibility)
			}
		}
	}
	if !sawSteer {
		t.Fatalf("missing session.steer in %+v", listed)
	}
}

func TestIntent_steerRejectsWhenIdle(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	ev := events.NewService(db)
	rc := &stubRunControl{}
	svc := session.NewService(db, nil, ev).WithRunControl(rc)

	blank, err := svc.Create(session.CreateRequest{SpaceID: "local", CreatedBy: "actor1", RepoRoot: "."})
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.Intent(blank.ID, session.IntentRequest{Action: "steer", Prompt: "nope"})
	if err == nil {
		t.Fatal("expected idle blank steer to fail")
	}
	if !errors.Is(err, session.ErrIntentRejected) || !strings.Contains(err.Error(), "nothing to steer") {
		t.Fatalf("err=%v", err)
	}

	seedRun(t, db, "run_steer_idle", "finished")
	bound, err := svc.Create(session.CreateRequest{RunID: "run_steer_idle", SpaceID: "local", CreatedBy: "actor1"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.Intent(bound.ID, session.IntentRequest{Action: "steer", Prompt: "nope"})
	if err == nil {
		t.Fatal("expected terminal-run steer to fail")
	}
	if !errors.Is(err, session.ErrIntentRejected) || !strings.Contains(err.Error(), "nothing to steer") {
		t.Fatalf("err=%v", err)
	}
	if rc.cancelCalls != 0 {
		t.Fatalf("cancelCalls=%d want 0", rc.cancelCalls)
	}
}
