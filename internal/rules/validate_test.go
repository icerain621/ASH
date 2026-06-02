package rules

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseAndValidate_featureDelivery(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "scenarios", "feature_delivery.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	res := ParseAndValidate(raw)
	if !res.OK {
		t.Fatalf("expected valid DSL, issues: %+v", res.Issues)
	}
	if res.Doc.Scenario.Name != "feature_delivery" {
		t.Fatalf("unexpected name: %s", res.Doc.Scenario.Name)
	}
}

func TestValidate_duplicateStepID(t *testing.T) {
	raw := []byte(`
version: "ash.rules/v0.1"
scenario:
  name: bad
  scenarioVersion: "1.0.0"
  steps:
    - { id: dup, role: PM, kind: llm, promptRef: p.md }
    - { id: dup, role: PM, kind: llm, promptRef: p2.md }
`)
	res := ParseAndValidate(raw)
	if res.OK {
		t.Fatal("expected validation failure")
	}
}

func TestValidate_gateUnknownStep(t *testing.T) {
	raw := []byte(`
version: "ash.rules/v0.1"
scenario:
  name: bad
  scenarioVersion: "1.0.0"
  gates:
    - { id: g1, when: "before.step.missing", blocking: true, check: { tool: git.status } }
  steps:
    - { id: s1, role: PM, kind: llm, promptRef: p.md }
`)
	res := ParseAndValidate(raw)
	if res.OK {
		t.Fatal("expected validation failure for unknown gate step")
	}
}

func TestEngine_EvaluateHooks(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "scenarios", "feature_delivery.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	res := ParseAndValidate(raw)
	eng := NewEngine(res.Doc)
	denied, reason := eng.EvaluateHooks("tool.called", map[string]any{
		"tool": "shell.exec",
		"risk": "danger",
	})
	if !denied || reason == "" {
		t.Fatalf("expected hook deny, got denied=%v reason=%q", denied, reason)
	}
}
