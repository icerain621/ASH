package memory

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ash-repwiki/ash/internal/agentexec"
	"github.com/ash-repwiki/ash/internal/events"
	"github.com/ash-repwiki/ash/internal/rules"
	"github.com/ash-repwiki/ash/internal/runs"
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
	runsSvc := runs.NewService(db, ev, loader, toolbus.DefaultBus()).WithAgentExecutor(agentexec.StaticExecutor{})
	return NewService(db, ev), ev, runsSvc
}

func repoWithMemoryEvidence(t *testing.T, issue string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte(issue+"\nMemory SSE citation evidence.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
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

func TestMemoryGovernanceEdgesAndTTL(t *testing.T) {
	svc, _, _ := newTestMemory(t)
	confidence := 0.92

	base, err := svc.CreateCandidate(CreateCandidateRequest{
		Layer:     "L1",
		Title:     "Stable test policy",
		Body:      "Prefer stable integration tests for delivery flows.",
		ScopeRepo: "ash",
		Evidence:  []EvidenceInput{{Kind: "file", Ref: "doc/testing.md"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Review(base.CandidateID, ReviewRequest{
		Decision:      "approve",
		Reason:        "baseline policy",
		PolicyProfile: "default",
		Confidence:    &confidence,
	}); err != nil {
		t.Fatal(err)
	}

	dup, err := svc.CreateCandidate(CreateCandidateRequest{
		Layer:     "L1",
		Title:     "Stable test policy",
		Body:      "Prefer stable integration tests for delivery flows.",
		ScopeRepo: "ash",
		Evidence:  []EvidenceInput{{Kind: "file", Ref: "doc/testing.md"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Review(dup.CandidateID, ReviewRequest{
		Decision:      "approve",
		Reason:        "same policy",
		PolicyProfile: "default",
	}); err != nil {
		t.Fatal(err)
	}

	var duplicateEdges []store.MemoryEdge
	if err := svc.db.Where("from_id = ? AND to_id = ? AND kind = ?", dup.CandidateID, base.CandidateID, "duplicate").Find(&duplicateEdges).Error; err != nil {
		t.Fatal(err)
	}
	if len(duplicateEdges) != 1 {
		t.Fatalf("duplicate edges=%d want 1", len(duplicateEdges))
	}

	q, err := svc.Query(QueryRequest{Text: "stable integration", TopK: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(q.Items) != 1 || q.Items[0].ID != base.CandidateID {
		t.Fatalf("query items=%+v want only base", q.Items)
	}
	if q.Items[0].Confidence != confidence {
		t.Fatalf("confidence=%v want %v", q.Items[0].Confidence, confidence)
	}

	replacementConfidence := 0.95
	replacement, err := svc.CreateCandidate(CreateCandidateRequest{
		Layer:     "L1",
		Title:     "Delivery test policy",
		Body:      "Prefer stable integration tests plus replay coverage.",
		ScopeRepo: "ash",
		Evidence:  []EvidenceInput{{Kind: "file", Ref: "doc/testing-v2.md"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Review(replacement.CandidateID, ReviewRequest{
		Decision:      "approve",
		Reason:        "supersedes baseline",
		PolicyProfile: "default",
		Confidence:    &replacementConfidence,
		Replaces:      []string{base.CandidateID},
	}); err != nil {
		t.Fatal(err)
	}

	var baseRecord store.MemoryRecord
	if err := svc.db.First(&baseRecord, "id = ?", base.CandidateID).Error; err != nil {
		t.Fatal(err)
	}
	if baseRecord.Status != "deprecated" {
		t.Fatalf("base status=%q want deprecated", baseRecord.Status)
	}
	var replacesEdges []store.MemoryEdge
	if err := svc.db.Where("from_id = ? AND to_id = ? AND kind = ?", replacement.CandidateID, base.CandidateID, "replaces").Find(&replacesEdges).Error; err != nil {
		t.Fatal(err)
	}
	if len(replacesEdges) != 1 {
		t.Fatalf("replaces edges=%d want 1", len(replacesEdges))
	}

	autoConflict, err := svc.CreateCandidate(CreateCandidateRequest{
		Layer:     "L1",
		Title:     "Delivery test policy",
		Body:      "Prefer smoke tests only and skip replay coverage.",
		ScopeRepo: "ash",
		Evidence:  []EvidenceInput{{Kind: "file", Ref: "doc/testing-fast.md"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Review(autoConflict.CandidateID, ReviewRequest{
		Decision:      "approve",
		Reason:        "same title but different guidance",
		PolicyProfile: "default",
	}); err != nil {
		t.Fatal(err)
	}
	var autoConflictEdges []store.MemoryEdge
	if err := svc.db.Where("from_id = ? AND to_id = ? AND kind = ?", autoConflict.CandidateID, replacement.CandidateID, "conflict").Find(&autoConflictEdges).Error; err != nil {
		t.Fatal(err)
	}
	if len(autoConflictEdges) != 1 || autoConflictEdges[0].Reason != "title matched approved memory with different body" {
		t.Fatalf("auto conflict edges=%+v want title/body conflict", autoConflictEdges)
	}

	conflict, err := svc.CreateCandidate(CreateCandidateRequest{
		Layer:     "L1",
		Title:     "Fast test policy",
		Body:      "Prefer smoke-only tests for delivery flows.",
		ScopeRepo: "ash",
		Evidence:  []EvidenceInput{{Kind: "file", Ref: "doc/testing-fast.md"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Review(conflict.CandidateID, ReviewRequest{
		Decision:      "approve",
		Reason:        "conflicts with replay coverage",
		PolicyProfile: "default",
		ConflictsWith: []string{replacement.CandidateID},
	}); err != nil {
		t.Fatal(err)
	}
	view, err := svc.Get(conflict.CandidateID)
	if err != nil {
		t.Fatal(err)
	}
	foundConflict := false
	for _, edge := range view.Edges {
		if edge.Kind == "conflict" && edge.ToID == replacement.CandidateID {
			foundConflict = true
		}
	}
	if !foundConflict {
		t.Fatalf("expected conflict edge in view: %+v", view.Edges)
	}

	ttl := 1
	expired, err := svc.CreateCandidate(CreateCandidateRequest{
		Layer:   "L0",
		Title:   "Expired local note",
		Body:    "temporary memory should expire",
		TTLDays: &ttl,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Review(expired.CandidateID, ReviewRequest{
		Decision:      "approve",
		Reason:        "temporary",
		PolicyProfile: "default",
	}); err != nil {
		t.Fatal(err)
	}
	old := time.Now().UTC().Add(-48 * time.Hour)
	if err := svc.db.Model(&store.MemoryRecord{}).Where("id = ?", expired.CandidateID).Updates(map[string]any{
		"created_at": old,
		"updated_at": old,
	}).Error; err != nil {
		t.Fatal(err)
	}
	q, err = svc.Query(QueryRequest{Text: "temporary memory", TopK: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(q.Items) != 0 {
		t.Fatalf("expired query items=%+v want none", q.Items)
	}
}

func TestMemoryEventsOnRunSSE(t *testing.T) {
	mem, ev, runsSvc := newTestMemory(t)

	run, err := runsSvc.Create(runs.CreateRequest{
		Scenario: runs.ScenarioRef{Name: "feature_delivery", ScenarioVersion: "1.0.0"},
		Inputs: map[string]any{
			"issueOrSpec": "memory sse test",
			"repoRoot":    repoWithMemoryEvidence(t, "memory sse test"),
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
		"memory.candidate_created": 0,
		"memory.review_requested":  0,
		"memory.reviewed":          0,
		"memory.hit_used":          0,
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

func TestCreateCandidateGovernanceHints(t *testing.T) {
	svc, _, _ := newTestMemory(t)
	confidence := 0.9
	base, err := svc.CreateCandidate(CreateCandidateRequest{
		Layer: "L1", Title: "Governance hint", Body: "Always run doctor before release.",
		ScopeRepo: "ash", Evidence: []EvidenceInput{{Kind: "file", Ref: "doc/hint.md"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Review(base.CandidateID, ReviewRequest{
		Decision: "approve", Reason: "baseline", PolicyProfile: "default", Confidence: &confidence,
	}); err != nil {
		t.Fatal(err)
	}
	dup, err := svc.CreateCandidate(CreateCandidateRequest{
		Layer: "L1", Title: "Governance hint", Body: "Always run doctor before release.",
		ScopeRepo: "ash", Evidence: []EvidenceInput{{Kind: "file", Ref: "doc/hint.md"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if dup.Governance == nil || len(dup.Governance.Duplicates) == 0 {
		t.Fatalf("governance=%+v want duplicate hint", dup.Governance)
	}
}
