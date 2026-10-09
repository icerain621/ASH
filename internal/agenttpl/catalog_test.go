package agenttpl

import (
	"strings"
	"testing"
)

func TestCustomTemplateRequiresApproval(t *testing.T) {
	ResetCatalogForTest()
	t.Cleanup(ResetCatalogForTest)
	m := Manifest{
		ID: "tpl.custom.demo", Version: "1.0.0", Loop: "react",
		Tools: []string{"read"}, Compaction: "threshold", Sandbox: "workspace-write",
		MaxTurns: 2, Memory: []string{"mem.retrieve", "mem.inject"},
	}
	entry, err := SubmitCandidate(m)
	if err != nil {
		t.Fatal(err)
	}
	if entry.Status != StatusCandidate {
		t.Fatalf("%+v", entry)
	}
	if _, err := GetProduction(m.ID); err == nil || !strings.Contains(err.Error(), "not approved") {
		t.Fatalf("err=%v", err)
	}
	if _, err := ApproveTemplate(m.ID); err != nil {
		t.Fatal(err)
	}
	got, err := GetProduction(m.ID)
	if err != nil || got.ID != m.ID {
		t.Fatalf("got=%+v err=%v", got, err)
	}
	if _, err := SubmitCandidate(m); err == nil {
		t.Fatal("must not demote approved template via SubmitCandidate")
	}
}
