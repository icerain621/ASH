package rules

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidate_verifyStepRequiresChecks(t *testing.T) {
	raw := []byte(`
version: "ash.rules/v0.1"
scenario:
  name: t
  scenarioVersion: "1.0.0"
  steps:
    - id: v1
      role: QA
      kind: verify
`)
	res := ParseAndValidate(raw)
	if res.OK {
		t.Fatal("expected invalid")
	}
}

func TestParseAndValidate_featureDeliveryIsThinTemplates(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "scenarios", "feature_delivery.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	res := ParseAndValidate(raw)
	if !res.OK {
		t.Fatalf("%v", res.Issues)
	}
	want := map[string]string{
		"plan.clarify":   "agent",
		"code.implement": "agent",
		"review.quality": "review",
	}
	got := map[string]string{}
	for _, st := range res.Doc.Scenario.Steps {
		got[st.ID] = st.Kind
		if st.Kind == "agent" || st.Kind == "review" {
			if st.Agent == nil || st.Agent.TemplateID == "" {
				t.Fatalf("step %s missing templateId", st.ID)
			}
		}
	}
	for id, kind := range want {
		if got[id] != kind {
			t.Fatalf("step %s kind=%q want %q (got=%v)", id, got[id], kind, got)
		}
	}
}
