package openapicheck

import (
	"path/filepath"
	"testing"
)

func TestValidateReadyzContract(t *testing.T) {
	root := repoRoot(t)
	if err := ValidateReadyzContract(root); err != nil {
		t.Fatal(err)
	}
}

func TestValidateReadyzContract_missingProperty(t *testing.T) {
	root := repoRoot(t)
	contractPath := filepath.Join(root, contractRelPath)
	names, err := SchemaPropertyNames(contractPath, "HealthResponse")
	if err != nil {
		t.Fatal(err)
	}
	if len(names) < len(readyzHealthRequiredProps) {
		t.Fatalf("HealthResponse properties=%d want >= %d", len(names), len(readyzHealthRequiredProps))
	}
}
