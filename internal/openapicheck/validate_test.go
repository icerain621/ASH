package openapicheck

import (
	"testing"
)

func TestValidateContract(t *testing.T) {
	root := repoRoot(t)
	rep, err := ValidateContract(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.MissingPaths) > 0 {
		t.Fatalf("missing=%v", rep.MissingPaths)
	}
	if len(rep.GenericEnvelope) > 0 {
		t.Fatalf("generic=%v", rep.GenericEnvelope)
	}
	if rep.LegacyPlanned != 0 {
		t.Fatalf("legacy planned=%d want 0", rep.LegacyPlanned)
	}
	if err := ValidateContractOrError(root); err != nil {
		t.Fatal(err)
	}
}
