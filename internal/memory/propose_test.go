package memory

import (
	"strings"
	"testing"
	"time"

	"github.com/ash-repwiki/ash/internal/events"
	"github.com/ash-repwiki/ash/internal/store"
)

func testMemoryService(t *testing.T) *Service {
	t.Helper()
	db := store.OpenTest(t, t.TempDir())
	return NewService(db, events.NewService(db))
}

func seedApproved(t *testing.T, svc *Service, title, body string) string {
	t.Helper()
	created, err := svc.CreateCandidate(CreateCandidateRequest{
		Layer: "L1", Title: title, Body: body,
		Evidence: []EvidenceInput{{Kind: "file", Ref: "doc/" + title + ".md"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Review(created.CandidateID, ReviewRequest{
		Decision: "approve", Reason: "ok", PolicyProfile: "default",
	}); err != nil {
		t.Fatal(err)
	}
	return created.CandidateID
}

func TestProposeRequiresEvidence(t *testing.T) {
	svc := testMemoryService(t)
	_, err := svc.Propose(ProposeRequest{CreateCandidateRequest: CreateCandidateRequest{
		Layer: "L0", Title: "t", Body: "b",
	}})
	if err == nil || !strings.Contains(err.Error(), "evidence") {
		t.Fatalf("err=%v", err)
	}
}

func TestProposeRejectsTrajectory(t *testing.T) {
	svc := testMemoryService(t)
	_, err := svc.Propose(ProposeRequest{CreateCandidateRequest: CreateCandidateRequest{
		Layer: "L0", Title: "t", Body: "b",
		Evidence: []EvidenceInput{{Kind: "trajectory", Ref: "run_1"}},
	}})
	if err == nil || !strings.Contains(err.Error(), "trajectory") {
		t.Fatalf("err=%v", err)
	}
}

func TestProposeCreatesCandidate(t *testing.T) {
	svc := testMemoryService(t)
	out, err := svc.Propose(ProposeRequest{CreateCandidateRequest: CreateCandidateRequest{
		Layer: "L0", Title: "proposed", Body: "body",
		Evidence: []EvidenceInput{{Kind: "file", Ref: "doc/a.md"}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if out.CandidateID == "" {
		t.Fatal("empty candidate")
	}
}

func TestConsolidateDoesNotMutateApproved(t *testing.T) {
	svc := testMemoryService(t)
	primary := seedApproved(t, svc, "primary note", "keep this body")
	dup := seedApproved(t, svc, "dup note", "other body")
	var before store.MemoryRecord
	if err := svc.gdb().First(&before, "id = ?", primary).Error; err != nil {
		t.Fatal(err)
	}
	out, err := svc.Consolidate(ConsolidateRequest{
		PrimaryID: primary, DuplicateIDs: []string{dup}, Reason: "merge dups",
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.CandidateID == "" {
		t.Fatal("empty candidate")
	}
	var after store.MemoryRecord
	if err := svc.gdb().First(&after, "id = ?", primary).Error; err != nil {
		t.Fatal(err)
	}
	if after.Body != before.Body || after.Status != "approved" {
		t.Fatalf("approved mutated: %+v", after)
	}
}

func TestForgetProposeLeavesApproved(t *testing.T) {
	svc := testMemoryService(t)
	id := seedApproved(t, svc, "forget me", "body")
	out, err := svc.ForgetPropose(ForgetRequest{MemoryID: id, Reason: "ttl expired"})
	if err != nil {
		t.Fatal(err)
	}
	var still store.MemoryRecord
	if err := svc.gdb().First(&still, "id = ?", id).Error; err != nil {
		t.Fatal(err)
	}
	if still.Status != "approved" {
		t.Fatalf("status=%q", still.Status)
	}
	if out.CandidateID == "" {
		t.Fatal("empty candidate")
	}
	_ = time.Now()
}

func TestKnowledgeReadOnly(t *testing.T) {
	svc := testMemoryService(t)
	created, err := svc.CreateCandidate(CreateCandidateRequest{
		Layer: "L2", Title: "wiki", Body: "projection",
		Evidence: []EvidenceInput{{Kind: "file", Ref: "doc/wiki.md"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Review(created.CandidateID, ReviewRequest{
		Decision: "approve", Reason: "ok", PolicyProfile: "default",
	}); err != nil {
		t.Fatal(err)
	}
	hits, err := svc.Knowledge("local", 10)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, h := range hits {
		if h.ID == created.CandidateID {
			found = true
		}
	}
	if !found {
		t.Fatalf("hits=%+v", hits)
	}
}
