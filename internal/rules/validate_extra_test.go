package rules

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScenarioRef(t *testing.T) {
	doc := &Document{Scenario: Scenario{Name: "n", ScenarioVersion: "1.2.3"}}
	ref := doc.Scenario.Ref()
	if ref.Name != "n" || ref.ScenarioVersion != "1.2.3" {
		t.Fatalf("%+v", ref)
	}
}

func TestValidateGateWhenVariants(t *testing.T) {
	yaml := `
apiVersion: ash.scenario/v1
kind: Scenario
scenario:
  name: gate_cases
  scenarioVersion: "0.0.1"
  policyProfile: default
  steps:
    - id: s1
      role: Coder
      kind: code
  gates:
    - when: before.step.
      require: []
`
	res := ParseAndValidate([]byte(yaml))
	if res.OK {
		t.Fatal("empty step ref should fail")
	}

	yaml = `
apiVersion: ash.scenario/v1
kind: Scenario
scenario:
  name: gate_role
  scenarioVersion: "0.0.1"
  policyProfile: default
  steps:
    - id: s1
      role: Coder
      kind: code
  gates:
    - when: before.role.Missing
      require: []
`
	res = ParseAndValidate([]byte(yaml))
	if res.OK {
		t.Fatal("unknown role gate should fail")
	}

	yaml = `
apiVersion: ash.scenario/v1
kind: Scenario
scenario:
  name: gate_bad_when
  scenarioVersion: "0.0.1"
  policyProfile: default
  steps:
    - id: s1
      role: Coder
      kind: code
  gates:
    - when: after.step.s1
      require: []
`
	res = ParseAndValidate([]byte(yaml))
	if res.OK {
		t.Fatal("unsupported when should fail")
	}

	yaml = `
apiVersion: ash.scenario/v1
kind: Scenario
scenario:
  name: gate_ok_role
  scenarioVersion: "0.0.1"
  policyProfile: default
  inputs:
    required: [""]
  steps:
    - id: s1
      role: Coder
      kind: code
  gates:
    - when: before.role.Coder
      require: []
    - when: run.before_finish
      require: []
`
	res = ParseAndValidate([]byte(yaml))
	if res.OK {
		t.Fatal("empty required input should fail")
	}
}

func TestParseYAMLInvalidAndLoaderErrors(t *testing.T) {
	if _, err := ParseYAML([]byte("{\ninvalid")); err == nil {
		t.Fatal("want parse error")
	}
	res := ParseAndValidate([]byte("{\ninvalid"))
	if res.OK {
		t.Fatal("want validate fail on bad yaml")
	}

	loader := NewLoader("")
	if err := loader.LoadDir(); err == nil {
		t.Fatal("empty dir config should fail")
	}
	loader = NewLoader(filepath.Join(t.TempDir(), "missing"))
	if err := loader.LoadDir(); err == nil {
		t.Fatal("missing dir should fail")
	}
	file := filepath.Join(t.TempDir(), "notadir")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	loader = NewLoader(file)
	if err := loader.LoadDir(); err == nil {
		t.Fatal("file path should fail")
	}

	badDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(badDir, "bad.yaml"), []byte("not: valid: scenario: [[["), 0o644); err != nil {
		t.Fatal(err)
	}
	loader = NewLoader(badDir)
	if err := loader.LoadDir(); err == nil || !strings.Contains(err.Error(), "scenario load errors") {
		t.Fatalf("want load errors, got %v", err)
	}

	_, err := loader.RawYAML("nope", "0")
	if err == nil {
		t.Fatal("raw missing should fail")
	}
}
