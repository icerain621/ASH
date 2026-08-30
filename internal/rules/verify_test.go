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

func TestParseAndValidate_featureDeliveryHasVerify(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "scenarios", "feature_delivery.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	res := ParseAndValidate(raw)
	if !res.OK {
		t.Fatalf("%v", res.Issues)
	}
	found := false
	for _, st := range res.Doc.Scenario.Steps {
		if st.ID == "qa.verify" && st.Kind == "verify" {
			found = true
			if st.Verify == nil || len(st.Verify.Checks) == 0 || st.Verify.OnFail != "improve" {
				t.Fatalf("%+v", st.Verify)
			}
		}
	}
	if !found {
		t.Fatal("qa.verify missing")
	}
}
