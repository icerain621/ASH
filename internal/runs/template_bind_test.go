package runs

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/ash-repwiki/ash/internal/agentexec"
	"github.com/ash-repwiki/ash/internal/agenttpl"
	"github.com/ash-repwiki/ash/internal/events"
	"github.com/ash-repwiki/ash/internal/plugins"
	"github.com/ash-repwiki/ash/internal/rules"
	"github.com/ash-repwiki/ash/internal/store"
	"github.com/ash-repwiki/ash/internal/toolbus"
)

func TestValidateProductionTemplates(t *testing.T) {
	agenttpl.ResetCatalogForTest()
	t.Cleanup(agenttpl.ResetCatalogForTest)

	doc := &rules.Document{Scenario: rules.Scenario{Steps: []rules.Step{
		{ID: "a", Kind: "agent", Agent: &rules.AgentSpec{TemplateID: "tpl.react"}},
		{ID: "r", Kind: "review"},
	}}}
	if err := validateProductionTemplates(doc); err != nil {
		t.Fatal(err)
	}

	m := agenttpl.Manifest{
		ID: "tpl.custom.prescan", Version: "1.0.0", Loop: "react",
		Tools: []string{"read"}, Compaction: "threshold", Sandbox: "workspace-write",
		MaxTurns: 2, Memory: []string{"mem.retrieve", "mem.inject"},
	}
	if _, err := agenttpl.SubmitCandidate(m); err != nil {
		t.Fatal(err)
	}
	bad := &rules.Document{Scenario: rules.Scenario{Steps: []rules.Step{
		{ID: "x", Kind: "agent", Agent: &rules.AgentSpec{TemplateID: m.ID}},
	}}}
	if err := validateProductionTemplates(bad); err == nil || !errors.Is(err, agenttpl.ErrNotApproved) {
		t.Fatalf("err=%v", err)
	}
	unknown := &rules.Document{Scenario: rules.Scenario{Steps: []rules.Step{
		{ID: "y", Kind: "agent", Agent: &rules.AgentSpec{TemplateID: "tpl.missing.zzz"}},
	}}}
	if err := validateProductionTemplates(unknown); err == nil || !errors.Is(err, agenttpl.ErrUnknown) {
		t.Fatalf("err=%v", err)
	}
}

func TestValidateProductionTemplates_MemoryBind(t *testing.T) {
	agenttpl.ResetCatalogForTest()
	t.Cleanup(agenttpl.ResetCatalogForTest)
	agenttpl.ClearApprovedLoaderForTest()
	t.Cleanup(agenttpl.ClearApprovedLoaderForTest)
	plugins.ResetGlobalForTest()
	t.Cleanup(plugins.ResetGlobalForTest)

	agenttpl.SetApprovedLoader(func(id string) (agenttpl.Manifest, bool) {
		if id != "tpl.from.db.inject" {
			return agenttpl.Manifest{}, false
		}
		return agenttpl.Manifest{
			ID: id, Version: "1.0.0", Loop: "react",
			Tools: []string{"read"}, Compaction: "threshold", Sandbox: "workspace-write",
			MaxTurns: 2, Memory: []string{"mem.inject"},
		}, true
	})
	doc := &rules.Document{Scenario: rules.Scenario{Steps: []rules.Step{
		{ID: "a", Kind: "agent", Agent: &rules.AgentSpec{TemplateID: "tpl.from.db.inject"}},
	}}}
	if err := validateProductionTemplates(doc); err == nil || !errors.Is(err, plugins.ErrInvalidBind) {
		t.Fatalf("err=%v", err)
	}

	if err := plugins.Global().Register(plugins.Component{
		ID: "mem.custom.gate", Kind: plugins.KindMemory, Status: plugins.StatusCandidate,
	}); err != nil {
		t.Fatal(err)
	}
	agenttpl.SetApprovedLoader(func(id string) (agenttpl.Manifest, bool) {
		if id != "tpl.from.db.mem" {
			return agenttpl.Manifest{}, false
		}
		return agenttpl.Manifest{
			ID: id, Version: "1.0.0", Loop: "react",
			Tools: []string{"read"}, Compaction: "threshold", Sandbox: "workspace-write",
			MaxTurns: 2, Memory: []string{"mem.custom.gate"},
		}, true
	})
	doc2 := &rules.Document{Scenario: rules.Scenario{Steps: []rules.Step{
		{ID: "b", Kind: "agent", Agent: &rules.AgentSpec{TemplateID: "tpl.from.db.mem"}},
	}}}
	if err := validateProductionTemplates(doc2); err == nil || !errors.Is(err, plugins.ErrNotApproved) {
		t.Fatalf("err=%v", err)
	}
}

func TestCreateRejectsUnapprovedTemplate(t *testing.T) {
	agenttpl.ResetCatalogForTest()
	t.Cleanup(agenttpl.ResetCatalogForTest)

	m := agenttpl.Manifest{
		ID: "tpl.custom.create.gate", Version: "1.0.0", Loop: "react",
		Tools: []string{"read"}, Compaction: "threshold", Sandbox: "workspace-write",
		MaxTurns: 2, Memory: []string{"mem.retrieve", "mem.inject"},
	}
	if _, err := agenttpl.SubmitCandidate(m); err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	scenarioYAML := `
version: "ash.rules/v0.1"
scenario:
  name: "custom_tpl_gate"
  scenarioVersion: "1.0.0"
  policyProfile: "default"
  roles:
    Coder: { maxParallel: 1 }
  inputs:
    required: [issueOrSpec]
  steps:
    - id: "only"
      role: "Coder"
      kind: "agent"
      agent:
        templateId: "tpl.custom.create.gate"
`
	if err := os.WriteFile(filepath.Join(dir, "custom_tpl_gate.yaml"), []byte(scenarioYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	db := store.OpenTest(t, t.TempDir())
	loader := rules.NewLoader(dir)
	if err := loader.LoadDir(); err != nil {
		t.Fatal(err)
	}
	svc := NewService(db, events.NewService(db), loader, toolbus.DefaultBus()).WithAgentExecutor(agentexec.StaticExecutor{})

	resp, err := svc.Create(CreateRequest{
		Scenario: ScenarioRef{Name: "custom_tpl_gate", ScenarioVersion: "1.0.0"},
		Inputs:   map[string]any{"issueOrSpec": "gate"},
	})
	if resp != nil {
		t.Fatalf("expected nil resp, got %+v", resp)
	}
	if err == nil || !errors.Is(err, agenttpl.ErrNotApproved) {
		t.Fatalf("err=%v", err)
	}
	var n int64
	if err := db.Model(&store.RunRecord{}).Count(&n).Error; err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("run rows=%d want 0", n)
	}
}
