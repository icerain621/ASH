package agenttpl

import (
	"strings"
	"testing"
)

func TestGetProductionUsesApprovedLoader(t *testing.T) {
	ResetCatalogForTest()
	t.Cleanup(ResetCatalogForTest)
	ClearApprovedLoaderForTest()
	t.Cleanup(ClearApprovedLoaderForTest)
	SetApprovedLoader(func(id string) (Manifest, bool) {
		if id != "tpl.from.db" {
			return Manifest{}, false
		}
		return Manifest{
			ID: id, Version: "1.0.0", Loop: "react", Tools: []string{"read"},
			Compaction: "threshold", Sandbox: "workspace-write", MaxTurns: 2,
			Memory: []string{"mem.retrieve", "mem.inject"},
		}, true
	})
	got, err := GetProduction("tpl.from.db")
	if err != nil || got.ID != "tpl.from.db" {
		t.Fatalf("got=%+v err=%v", got, err)
	}
}

func TestListEntriesIncludesStatus(t *testing.T) {
	ResetCatalogForTest()
	t.Cleanup(ResetCatalogForTest)
	m := Manifest{
		ID: "tpl.custom.list", Version: "1.0.0", Loop: "react",
		Tools: []string{"read"}, Compaction: "threshold", Sandbox: "workspace-write",
		MaxTurns: 2, Memory: []string{"mem.retrieve", "mem.inject"},
	}
	if _, err := SubmitCandidate(m); err != nil {
		t.Fatal(err)
	}
	entries, err := ListEntries()
	if err != nil {
		t.Fatal(err)
	}
	var found *Entry
	for i := range entries.Items {
		if entries.Items[i].Manifest.ID == m.ID {
			found = &entries.Items[i]
			break
		}
	}
	if found == nil || found.Status != StatusCandidate {
		t.Fatalf("candidate missing or wrong status: %+v", found)
	}
	if IsBindable(m.ID) {
		t.Fatal("candidate must not be bindable")
	}
	if !IsBindable("tpl.react") {
		t.Fatal("built-in must be bindable")
	}
}

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
