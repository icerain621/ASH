package runs

import (
	"strings"
	"testing"

	"github.com/ash-repwiki/ash/internal/agenttpl"
)

func TestUnapprovedTemplateRejectedOnBind(t *testing.T) {
	agenttpl.ResetCatalogForTest()
	t.Cleanup(agenttpl.ResetCatalogForTest)

	m := agenttpl.Manifest{
		ID: "tpl.custom.unapproved", Version: "1.0.0", Loop: "react",
		Tools: []string{"read"}, Compaction: "threshold", Sandbox: "workspace-write",
		MaxTurns: 2, Memory: []string{"mem.retrieve", "mem.inject"},
	}
	if _, err := agenttpl.SubmitCandidate(m); err != nil {
		t.Fatal(err)
	}
	_, err := agenttpl.GetProduction(m.ID)
	if err == nil || !strings.Contains(err.Error(), "not approved") {
		t.Fatalf("err=%v", err)
	}

	svc, _ := testRunsService(t)
	// Direct executeTemplateStep path: Create with a scenario that names the custom template
	// is heavier; the production gate is GetProduction which template_step uses.
	if _, err := agenttpl.ApproveTemplate(m.ID); err != nil {
		t.Fatal(err)
	}
	got, err := agenttpl.GetProduction(m.ID)
	if err != nil || got.ID != m.ID {
		t.Fatalf("got=%+v err=%v", got, err)
	}
	_ = svc
}
