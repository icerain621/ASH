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

func setRunStatus(t *testing.T, db *store.DB, id, status string) {
	t.Helper()
	if err := db.Model(&store.RunRecord{}).Where("id = ?", id).Update("status", status).Error; err != nil {
		t.Fatal(err)
	}
}

func followUpPrompts(meta map[string]any) []string {
	if meta == nil {
		return nil
	}
	switch v := meta["followUpQueue"].(type) {
	case []string:
		return append([]string(nil), v...)
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

func TestIntent_queueIdleActsAsPrompt(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	ev := events.NewService(db)
	svc := session.NewService(db, nil, ev)
	view, err := svc.Create(session.CreateRequest{SpaceID: "local", CreatedBy: "actor1", RepoRoot: "."})
	if err != nil {
		t.Fatal(err)
	}
	out, err := svc.Intent(view.ID, session.IntentRequest{Action: "queue", Prompt: "do it now"})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Turns) != 1 || out.Turns[0].Prompt != "do it now" {
		t.Fatalf("turns=%+v", out.Turns)
	}
	if prompts := followUpPrompts(out.Meta); len(prompts) != 0 {
		t.Fatalf("queue=%v want empty", prompts)
	}
}

func TestIntent_queueWhileRunActiveEnqueuesAndDoesNotSteer(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	ev := events.NewService(db)
	rc := &stubRunControl{}
	svc := session.NewService(db, nil, ev).WithRunControl(rc)
	seedRun(t, db, "run_q_1", "running")
	view, err := svc.Create(session.CreateRequest{RunID: "run_q_1", SpaceID: "local", CreatedBy: "actor1"})
	if err != nil {
		t.Fatal(err)
	}

	out, err := svc.Intent(view.ID, session.IntentRequest{Action: "queue", Prompt: "after this", ActorID: "actor1"})
	if err != nil {
		t.Fatal(err)
	}
	if rc.cancelCalls != 0 {
		t.Fatalf("queue must not cancel, cancelCalls=%d", rc.cancelCalls)
	}
	if len(out.Turns) != 0 {
		t.Fatalf("turns=%+v want none while running", out.Turns)
	}
	got, err := svc.Get(view.ID)
	if err != nil {
		t.Fatal(err)
	}
	prompts := followUpPrompts(got.Meta)
	if len(prompts) != 1 || prompts[0] != "after this" {
		t.Fatalf("queue=%v", prompts)
	}

	listed, err := ev.ListAfter("run_q_1", 0, 50)
	if err != nil {
		t.Fatal(err)
	}
	saw := false
	for _, item := range listed {
		if item.Type != "session.queue" {
			continue
		}
		saw = true
		if !strings.Contains(string(item.Payload), "after this") || !strings.Contains(string(item.Payload), "enqueue") {
			t.Fatalf("payload=%s", item.Payload)
		}
	}
	if !saw {
		t.Fatal("missing session.queue")
	}

	_, err = svc.Intent(view.ID, session.IntentRequest{Action: "stop", ActorID: "actor1"})
	if err != nil {
		t.Fatal(err)
	}
	if rc.cancelCalls != 1 {
		t.Fatalf("stop cancelCalls=%d", rc.cancelCalls)
	}
	got, err = svc.Get(view.ID)
	if err != nil {
		t.Fatal(err)
	}
	prompts = followUpPrompts(got.Meta)
	if len(prompts) != 1 || prompts[0] != "after this" {
		t.Fatalf("stop cleared queue=%v", prompts)
	}
	if len(got.Turns) != 0 {
		t.Fatalf("stop started turns=%+v", got.Turns)
	}
}

func TestIntent_queueDrainsOneAfterRunFinishesInOrder(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	ev := events.NewService(db)
	svc := session.NewService(db, nil, ev)
	seedRun(t, db, "run_q_2", "running")
	view, err := svc.Create(session.CreateRequest{RunID: "run_q_2", SpaceID: "local", CreatedBy: "actor1"})
	if err != nil {
		t.Fatal(err)
	}
	for _, prompt := range []string{"first", "second"} {
		if _, err := svc.Intent(view.ID, session.IntentRequest{Action: "queue", Prompt: prompt}); err != nil {
			t.Fatal(err)
		}
	}

	mid, _, err := svc.PromptTurn(view.ID, session.TurnRequest{Prompt: "still running"})
	if err != nil {
		t.Fatal(err)
	}
	if len(mid.Turns) != 1 || mid.Turns[0].Prompt != "still running" {
		t.Fatalf("turns=%+v", mid.Turns)
	}
	if prompts := followUpPrompts(mid.Meta); len(prompts) != 2 {
		t.Fatalf("queue drained while run active: %v", prompts)
	}

	setRunStatus(t, db, "run_q_2", "canceled")
	svc.DrainFollowUpForRun("run_q_2")
	held, err := svc.Get(view.ID)
	if err != nil {
		t.Fatal(err)
	}
	if prompts := followUpPrompts(held.Meta); len(prompts) != 2 {
		t.Fatalf("cancel drain queue=%v", prompts)
	}

	setRunStatus(t, db, "run_q_2", "finished")
	svc.DrainFollowUpForRun("run_q_2")
	done, err := svc.Get(view.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(done.Turns) != 3 {
		t.Fatalf("turns=%d %+v", len(done.Turns), done.Turns)
	}
	if done.Turns[1].Prompt != "first" || done.Turns[2].Prompt != "second" {
		t.Fatalf("order=%q %q", done.Turns[1].Prompt, done.Turns[2].Prompt)
	}
	if prompts := followUpPrompts(done.Meta); len(prompts) != 0 {
		t.Fatalf("queue=%v", prompts)
	}
}

func TestIntent_queueRejectsEmptyPrompt(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	svc := session.NewService(db, nil, events.NewService(db))
	view, err := svc.Create(session.CreateRequest{SpaceID: "local", CreatedBy: "actor1", RepoRoot: "."})
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.Intent(view.ID, session.IntentRequest{Action: "queue", Prompt: "  "})
	if err == nil || !errors.Is(err, session.ErrIntentRejected) {
		t.Fatalf("err=%v", err)
	}
}

func TestIntent_steerDoesNotDeleteQueue(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	ev := events.NewService(db)
	rc := &stubRunControl{}
	svc := session.NewService(db, nil, ev).WithRunControl(rc)
	seedRun(t, db, "run_q_steer", "running")
	view, err := svc.Create(session.CreateRequest{RunID: "run_q_steer", SpaceID: "local", CreatedBy: "actor1"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Intent(view.ID, session.IntentRequest{Action: "queue", Prompt: "queued"}); err != nil {
		t.Fatal(err)
	}
	out, err := svc.Intent(view.ID, session.IntentRequest{Action: "steer", Prompt: "interrupt"})
	if err != nil {
		t.Fatal(err)
	}
	if rc.cancelCalls != 1 {
		t.Fatalf("cancelCalls=%d", rc.cancelCalls)
	}
	if len(out.Turns) != 1 || out.Turns[0].Prompt != "interrupt" {
		t.Fatalf("turns=%+v", out.Turns)
	}
	// Stub cancel does not flip run status, so the run stays active and the queue is not drained.
	if prompts := followUpPrompts(out.Meta); len(prompts) != 1 || prompts[0] != "queued" {
		t.Fatalf("queue=%v", prompts)
	}
}
