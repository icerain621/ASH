package memory

import (
	"testing"

	"github.com/ash-repwiki/ash/internal/store"
)

func TestFilterBySensitivityDropsOverClearance(t *testing.T) {
	rows := []store.MemoryRecord{
		{ID: "n", Sensitivity: "normal"},
		{ID: "r", Sensitivity: "restricted"},
		{ID: "s", Sensitivity: "secret"},
	}
	got := FilterBySensitivity(rows, ClearanceNormal)
	if len(got) != 1 || got[0].ID != "n" {
		t.Fatalf("visible=%v", ids(got))
	}
	all := FilterBySensitivity(rows, "secret")
	if len(all) != 3 {
		t.Fatalf("secret clearance=%v", ids(all))
	}
}

func TestQueryForSpaceHidesSecretFromNormalCaller(t *testing.T) {
	svc, _ := newTestMemory(t)
	secret := CreateCandidateRequest{
		Layer: "L0", Title: "secret token", Body: "do not leak",
		Sensitivity: "secret",
	}
	created, err := svc.CreateCandidate(secret)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Review(created.CandidateID, ReviewRequest{
		Decision: "approve", ReviewerID: "rev", Reason: "ok", PolicyProfile: "default",
	}); err != nil {
		t.Fatal(err)
	}
	normal := CreateCandidateRequest{Layer: "L0", Title: "secret token public", Body: "safe note"}
	pub, err := svc.CreateCandidate(normal)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Review(pub.CandidateID, ReviewRequest{
		Decision: "approve", ReviewerID: "rev", Reason: "ok", PolicyProfile: "default",
	}); err != nil {
		t.Fatal(err)
	}
	resp, err := svc.Query(QueryRequest{Text: "secret token", Clearance: ClearanceNormal})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Items) != 1 || resp.Items[0].ID != pub.CandidateID {
		t.Fatalf("items=%v", resp.Items)
	}
}

func ids(rows []store.MemoryRecord) []string {
	out := make([]string, len(rows))
	for i, row := range rows {
		out[i] = row.ID
	}
	return out
}
