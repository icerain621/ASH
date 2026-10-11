package runs

import (
	"errors"
	"testing"

	"github.com/ash-repwiki/ash/internal/agenttpl"
	"github.com/ash-repwiki/ash/internal/plugins"
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
	if err == nil || !errors.Is(err, agenttpl.ErrNotApproved) {
		t.Fatalf("err=%v", err)
	}
	if code := agentErrorCode(err); code != "AGENT_TEMPLATE_NOT_APPROVED" {
		t.Fatalf("code=%s", code)
	}
	if code := agentErrorCode(agenttpl.ErrUnknown); code != "AGENT_TEMPLATE_UNKNOWN" {
		t.Fatalf("unknown code=%s", code)
	}
	if code := agentErrorCode(plugins.ErrNotApproved); code != "AGENT_COMPONENT_NOT_APPROVED" {
		t.Fatalf("component code=%s", code)
	}
	if code := agentErrorCode(plugins.ErrInvalidBind); code != "AGENT_COMPONENT_BIND_INVALID" {
		t.Fatalf("bind invalid code=%s", code)
	}

	if _, err := agenttpl.ApproveTemplate(m.ID); err != nil {
		t.Fatal(err)
	}
	got, err := agenttpl.GetProduction(m.ID)
	if err != nil || got.ID != m.ID {
		t.Fatalf("got=%+v err=%v", got, err)
	}
}
