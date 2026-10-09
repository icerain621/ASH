package rules

import "testing"

func TestAdaptLegacyMapsOldKinds(t *testing.T) {
	doc := &Document{Version: Version, Scenario: Scenario{
		Name: "legacy", ScenarioVersion: "1.0.0",
		Steps: []Step{
			{ID: "a", Role: "PM", Kind: "llm", PromptRef: "p.md"},
			{ID: "b", Role: "Coder", Kind: "tool_chain", Chain: []ToolChainItem{{Tool: "git.status"}}},
			{ID: "c", Role: "QA", Kind: "verify", Verify: &VerifySpec{Checks: []ToolChainItem{{Tool: "test.run"}}}},
			{ID: "d", Role: "Coder", Kind: "agent", Agent: &AgentSpec{TemplateID: "tpl.react"}},
		},
	}}
	deps := AdaptLegacy(doc)
	if len(deps) != 3 {
		t.Fatalf("deps=%d want 3", len(deps))
	}
	if doc.Scenario.Steps[0].Kind != "agent" || doc.Scenario.Steps[0].Agent.TemplateID != "tpl.plan-solve" {
		t.Fatalf("llm adapt=%+v", doc.Scenario.Steps[0])
	}
	if doc.Scenario.Steps[1].Kind != "tool_chain" || doc.Scenario.Steps[2].Kind != "verify" {
		t.Fatalf("chain/verify should stay executable: %+v %+v", doc.Scenario.Steps[1], doc.Scenario.Steps[2])
	}
	if doc.Scenario.Steps[3].Kind != "agent" || doc.Scenario.Steps[3].Agent.TemplateID != "tpl.react" {
		t.Fatalf("thin step mutated=%+v", doc.Scenario.Steps[3])
	}
}
