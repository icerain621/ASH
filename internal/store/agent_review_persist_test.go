package store_test

import (
	"testing"

	"github.com/ash-repwiki/ash/internal/store"
)

func TestAgentTemplateAndReviewItemPersist(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	row, err := db.SaveAgentTemplateCandidate("local", "tpl.custom.persist", "1.0.0", `{"id":"tpl.custom.persist"}`, "tester")
	if err != nil {
		t.Fatal(err)
	}
	if row.Status != "candidate" {
		t.Fatalf("%+v", row)
	}
	if _, err := db.GetApprovedAgentTemplate("local", "tpl.custom.persist"); err == nil {
		t.Fatal("candidate must not resolve as approved")
	}
	approved, err := db.ApproveAgentTemplate("local", "tpl.custom.persist", "reviewer")
	if err != nil || approved.Status != "approved" {
		t.Fatalf("approved=%+v err=%v", approved, err)
	}
	got, err := db.GetApprovedAgentTemplate("local", "tpl.custom.persist")
	if err != nil || got.TemplateID != "tpl.custom.persist" {
		t.Fatalf("got=%+v err=%v", got, err)
	}
	if _, err := db.SaveAgentTemplateCandidate("local", "tpl.custom.persist", "1.0.0", `{}`, "tester"); err == nil {
		t.Fatal("must not demote approved")
	}

	item, err := db.CreateReviewItem("local", "agent_template", "tpl.custom.persist", "check")
	if err != nil {
		t.Fatal(err)
	}
	decided, err := db.DecideReviewItem(item.ID, "approve", "reviewer", "ok")
	if err != nil || decided.Status != "approved" {
		t.Fatalf("decided=%+v err=%v", decided, err)
	}
}
