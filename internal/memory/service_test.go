package memory

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/ash-repwiki/ash/internal/events"
	"github.com/ash-repwiki/ash/internal/runs"
	"github.com/ash-repwiki/ash/internal/rules"
	"github.com/ash-repwiki/ash/internal/store"
	"github.com/ash-repwiki/ash/internal/toolbus"
)

func newTestMemory(t *testing.T) (*Service, *events.Service, *runs.Service) {
	t.Helper()
	dir := t.TempDir()
	db, err := store.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	ev := events.NewService(db)
	scenariosDir := filepath.Join("..", "..", "scenarios")
	loader := rules.NewLoader(scenariosDir)
	if err := loader.LoadDir(); err != nil {
		t.Fatalf("load scenarios: %v", err)
	}
	runsSvc := runs.NewService(db, ev, loader, toolbus.DefaultBus())
	return NewService(db, ev), ev, runsSvc
}

func TestCandidateReviewFlow(t *testing.T) {
	svc, _, _ := newTestMemory(t)

	_, err := svc.CreateCandidate(CreateCandidateRequest{
		Layer: "L0",
		Title: "note",
		Body:  "body",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = svc.CreateCandidate(CreateCandidateRequest{
		Layer: "L1",
		Title: "needs evidence",
		Body:  "x",
	})
	if err == nil {
		t.Fatal("expected L1 without evidence to fail")
	}

	created, err := svc.CreateCandidate(CreateCandidateRequest{
		Layer: "L1",
		Title: "with evidence",
		Body:  "rule text",
		Evidence: []EvidenceInput{{
			Kind: "file",
			Ref:  "docs/README.md",
		}},
		ScopeRepo: "ash",
	})
	if err != nil {
		t.Fatal(err)
	}

	list, err := svc.ListCandidates("", "candidate", "", 10, 0)
	if err != nil || len(list.Items) < 2 {
		t.Fatalf("list: %v items=%d", err, len(list.Items))
	}

	rev, err := svc.Review(created.CandidateID, ReviewRequest{
		Decision:      "approve",
		Reason:        "looks good",
		PolicyProfile: "default",
		ReviewerID:    "tester",
	})
	if err != nil || !rev.OK || rev.Status != "approved" {
		t.Fatalf("review: %+v err=%v", rev, err)
	}

	q, err := svc.Query(QueryRequest{Text: "rule", TopK: 5})
	if err != nil || len(q.Items) != 1 {
		t.Fatalf("query: %+v err=%v", q, err)
	}

	_, err = svc.HitUsed(HitUsedRequest{
		RunID:     "run_test",
		RecordIDs: []string{created.CandidateID},
	})
	if !errors.Is(err, ErrRunNotFound) {
		t.Fatalf("expected ErrRunNotFound, got %v", err)
	}
}

func TestRejectCandidate(t *testing.T) {
	svc, _, _ := newTestMemory(t)
	created, _ := svc.CreateCandidate(CreateCandidateRequest{Layer: "L0", Title: "t", Body: "b"})
	rev, err := svc.Review(created.CandidateID, ReviewRequest{
		Decision: "reject", Reason: "no", PolicyProfile: "default",
	})
	if err != nil || rev.Status != "rejected" {
		t.Fatalf("reject: %+v err=%v", rev, err)
	}
	_, err = svc.Query(QueryRequest{Text: "t"})
	if err != nil {
		t.Fatal(err)
	}
}

func TestMemoryEventsOnRunSSE(t *testing.T) {
	mem, ev, runsSvc := newTestMemory(t)

	run, err := runsSvc.Create(runs.CreateRequest{
		Scenario: runs.ScenarioRef{Name: "feature_delivery", ScenarioVersion: "1.0.0"},
		Inputs: map[string]any{
			"issueOrSpec": "memory sse test",
			"repoRoot":    t.TempDir(),
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	created, err := mem.CreateCandidate(CreateCandidateRequest{
		RunID: run.RunID,
		Layer: "L1",
		Title: "sse rule",
		Body:  "emit on stream",
		Evidence: []EvidenceInput{{
			Kind: "file",
			Ref:  "README.md",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = mem.Review(created.CandidateID, ReviewRequest{
		RunID:         run.RunID,
		Decision:      "approve",
		Reason:        "ok",
		PolicyProfile: "default",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = mem.HitUsed(HitUsedRequest{
		RunID:     run.RunID,
		RecordIDs: []string{created.CandidateID},
	})
	if err != nil {
		t.Fatal(err)
	}

	evs, err := ev.ListAfter(run.RunID, 0, 500)
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]int{
		"memory.candidate_created":  0,
		"memory.review_requested":   0,
		"memory.reviewed":           0,
		"memory.hit_used":           0,
	}
	for _, e := range evs {
		if _, ok := want[e.Type]; ok {
			want[e.Type]++
		}
	}
	for typ, n := range want {
		if n == 0 {
			t.Fatalf("missing SSE event %s in run stream", typ)
		}
	}
}
