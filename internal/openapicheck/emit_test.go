package openapicheck

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestEmitContractBackfill(t *testing.T) {
	if os.Getenv("OPENAPI_EMIT") != "1" {
		t.Skip("set OPENAPI_EMIT=1 to print missing contract paths")
	}
	root := repoRoot(t)
	contract, err := LoadPathMethods(filepath.Join(root, "doc/api/openapi-ash-v1.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	swagger, err := LoadSwaggerOperations(filepath.Join(root, "internal/api/docs/swagger.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	fmt.Print(EmitMissingContractYAML(contract, swagger))
}
