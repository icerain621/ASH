package runs

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/ash-repwiki/ash/internal/agentexec"
	"github.com/ash-repwiki/ash/internal/store"
)

func TestCanTransitionMatrix(t *testing.T) {
	cases := []struct {
		from string
		to   string
		ok   bool
	}{
		{StatusRunning, StatusWaitingApproval, true},
		{StatusRunning, StatusFinished, true},
		{StatusRunning, StatusFailed, true},
		{StatusRunning, StatusCanceled, true},
		{StatusWaitingApproval, StatusRunning, true},
		{StatusWaitingApproval, StatusCanceled, true},
		{StatusFailed, StatusRunning, true},
		{StatusRunning, StatusRunning, false},
		{StatusFinished, StatusRunning, false},
		{StatusFinished, StatusFailed, false},
		{StatusCanceled, StatusFailed, false},
		{StatusCanceled, StatusFinished, false},
		{StatusCanceled, StatusRunning, false},
		{StatusWaitingApproval, StatusFinished, false},
		{StatusWaitingApproval, StatusFailed, false},
		{StatusFailed, StatusCanceled, false},
		{StatusFinished, StatusCanceled, false},
	}
	for _, tc := range cases {
		got := canTransition(tc.from, tc.to)
		if got != tc.ok {
			t.Fatalf("canTransition(%q,%q)=%v want %v", tc.from, tc.to, got, tc.ok)
		}
	}
}

func TestApplyRunStatusIllegal(t *testing.T) {
	rec := &store.RunRecord{Status: StatusCanceled}
	if err := applyRunStatus(rec, StatusFailed); !errors.Is(err, ErrIllegalStatusTransition) {
		t.Fatalf("err=%v want ErrIllegalStatusTransition", err)
	}
	if rec.Status != StatusCanceled {
		t.Fatalf("status=%q want unchanged canceled", rec.Status)
	}
}

func TestCancelIdempotentAndFromRunning(t *testing.T) {
	svc, _ := testRunsService(t)
	created, err := svc.Create(CreateRequest{
		Scenario: ScenarioRef{Name: "feature_delivery", ScenarioVersion: "1.0.0"},
		Inputs: map[string]any{
			"issueOrSpec": "cancel idempotent",
			"repoRoot":    repoWithEvidence(t, "cancel idempotent"),
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Force running then cancel (Create may already have finished).
	var rec store.RunRecord
	if err := svc.db.First(&rec, "id = ?", created.RunID).Error; err != nil {
		t.Fatal(err)
	}
	rec.Status = StatusRunning
	rec.FinishedAt = nil
	if err := svc.db.Save(&rec).Error; err != nil {
		t.Fatal(err)
	}

	first, err := svc.Cancel(created.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if first.Status != StatusCanceled {
		t.Fatalf("first cancel status=%q want canceled", first.Status)
	}
	second, err := svc.Cancel(created.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if second.Status != StatusCanceled {
		t.Fatalf("second cancel status=%q want canceled", second.Status)
	}

	for _, terminal := range []string{StatusFinished, StatusFailed} {
		rec.Status = terminal
		now := time.Now().UTC()
		rec.FinishedAt = &now
		if err := svc.db.Save(&rec).Error; err != nil {
			t.Fatal(err)
		}
		resp, err := svc.Cancel(created.RunID)
		if err != nil {
			t.Fatal(err)
		}
		if resp.Status != terminal {
			t.Fatalf("cancel on %s status=%q want unchanged", terminal, resp.Status)
		}
	}
}

func TestFailRunDoesNotOverwriteCanceled(t *testing.T) {
	svc, _ := testRunsService(t)
	created, err := svc.Create(CreateRequest{
		Scenario: ScenarioRef{Name: "feature_delivery", ScenarioVersion: "1.0.0"},
		Inputs: map[string]any{
			"issueOrSpec": "fail after cancel",
			"repoRoot":    repoWithEvidence(t, "fail after cancel"),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	var rec store.RunRecord
	if err := svc.db.First(&rec, "id = ?", created.RunID).Error; err != nil {
		t.Fatal(err)
	}
	rec.Status = StatusCanceled
	now := time.Now().UTC()
	rec.FinishedAt = &now
	if err := svc.db.Save(&rec).Error; err != nil {
		t.Fatal(err)
	}

	_, err = svc.failRun(&rec, created.RunID, rec.TraceID, now.Add(-time.Minute), "TEST_FAIL", "should not overwrite")
	if err == nil {
		t.Fatal("failRun should still return an error")
	}
	var after store.RunRecord
	if err := svc.db.First(&after, "id = ?", created.RunID).Error; err != nil {
		t.Fatal(err)
	}
	if after.Status != StatusCanceled {
		t.Fatalf("status=%q want canceled (not overwritten by failed)", after.Status)
	}
	if after.ErrorCode == "TEST_FAIL" {
		t.Fatalf("errorCode unexpectedly set on canceled run")
	}
}

func TestObserveCanceled(t *testing.T) {
	svc, _ := testRunsService(t)
	created, err := svc.Create(CreateRequest{
		Scenario: ScenarioRef{Name: "feature_delivery", ScenarioVersion: "1.0.0"},
		Inputs: map[string]any{
			"issueOrSpec": "observe cancel",
			"repoRoot":    repoWithEvidence(t, "observe cancel"),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	var rec store.RunRecord
	if err := svc.db.First(&rec, "id = ?", created.RunID).Error; err != nil {
		t.Fatal(err)
	}
	if err := svc.observeCanceled(&rec); err != nil {
		t.Fatalf("finished run should not look canceled: %v", err)
	}
	rec.Status = StatusCanceled
	if err := svc.db.Save(&rec).Error; err != nil {
		t.Fatal(err)
	}
	rec.Status = StatusRunning // stale in-memory
	if err := svc.observeCanceled(&rec); !errors.Is(err, ErrRunCanceled) {
		t.Fatalf("err=%v want ErrRunCanceled", err)
	}
	if rec.Status != StatusCanceled {
		t.Fatalf("status=%q want refreshed canceled", rec.Status)
	}
}

func TestMidLoopCancelStopsWithoutFinish(t *testing.T) {
	svc, _ := testRunsService(t)
	hold := &holdAgentExecutor{
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
	svc.WithAgentExecutor(hold)

	type result struct {
		resp *CreateResponse
		err  error
	}
	done := make(chan result, 1)
	go func() {
		resp, err := svc.Create(CreateRequest{
			Scenario: ScenarioRef{Name: "feature_delivery", ScenarioVersion: "1.0.0"},
			Inputs: map[string]any{
				"issueOrSpec": "mid-loop cancel",
				"repoRoot":    repoWithEvidence(t, "mid-loop cancel"),
			},
		})
		done <- result{resp: resp, err: err}
	}()

	select {
	case <-hold.started:
	case <-time.After(15 * time.Second):
		t.Fatal("timed out waiting for agent step to start")
	}

	var rec store.RunRecord
	deadline := time.Now().Add(5 * time.Second)
	for {
		if err := svc.db.Order("created_at desc").First(&rec).Error; err == nil && rec.Status == StatusRunning {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for running run row")
		}
		time.Sleep(20 * time.Millisecond)
	}

	canceled, err := svc.Cancel(rec.ID)
	if err != nil {
		t.Fatal(err)
	}
	if canceled.Status != StatusCanceled {
		t.Fatalf("cancel status=%q want canceled", canceled.Status)
	}

	select {
	case got := <-done:
		if got.err != nil {
			t.Fatalf("Create after mid-loop cancel err=%v want nil (terminal canceled)", got.err)
		}
		if got.resp == nil || got.resp.RunID != rec.ID {
			t.Fatalf("resp=%+v want runId %s", got.resp, rec.ID)
		}
	case <-time.After(15 * time.Second):
		t.Fatal("timed out waiting for Create to return after cancel")
	}

	var after store.RunRecord
	if err := svc.db.First(&after, "id = ?", rec.ID).Error; err != nil {
		t.Fatal(err)
	}
	if after.Status != StatusCanceled {
		t.Fatalf("status=%q want canceled (not finished/failed)", after.Status)
	}
}

// holdAgentExecutor blocks in Execute until Cancel closes release (or test closes it).
type holdAgentExecutor struct {
	agentexec.StaticExecutor
	started chan struct{}
	release chan struct{}
	once    sync.Once
}

func (e *holdAgentExecutor) Execute(ctx context.Context, req agentexec.Request) (*agentexec.Result, error) {
	select {
	case <-e.started:
	default:
		close(e.started)
	}
	select {
	case <-e.release:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	return e.StaticExecutor.Execute(ctx, req)
}

func (e *holdAgentExecutor) Cancel(ctx context.Context, taskID string) error {
	e.once.Do(func() { close(e.release) })
	return nil
}
