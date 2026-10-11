package rules

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseAndValidate_verifyReplanScenario(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "scenarios", "verify_replan.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	res := ParseAndValidate(raw)
	if !res.OK {
		t.Fatalf("%v", res.Issues)
	}
	var found bool
	for _, st := range res.Doc.Scenario.Steps {
		if st.ID != "qa.verify" {
			continue
		}
		found = true
		if st.Verify == nil || st.Verify.OnFail != "replan" || st.Verify.MaxReplanTurns != 2 {
			t.Fatalf("verify=%+v", st.Verify)
		}
	}
	if !found {
		t.Fatal("qa.verify missing")
	}
}

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
