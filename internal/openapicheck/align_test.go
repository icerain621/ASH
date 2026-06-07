package openapicheck

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func TestContractMatchesSwagger(t *testing.T) {
	root := repoRoot(t)
	contract, err := LoadPathMethods(filepath.Join(root, "doc/api/openapi-ash-v1.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	swagger, err := LoadPathMethods(filepath.Join(root, "internal/api/docs/swagger.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	rep := AlignContract(contract, swagger, "/api/v1/", "/v1/")
	if len(rep.Missing) > 0 {
		t.Fatalf("hand-written /api/v1 contract not satisfied by swag output:\n%s", strings.Join(rep.Missing, "\n"))
	}
	if len(rep.LegacyPlanned) == 0 {
		t.Fatal("expected legacy /v1 planned paths in contract draft")
	}
	if len(rep.Undocumented) > 0 {
		t.Fatalf("contract missing %d implemented /api/v1 ops", len(rep.Undocumented))
	}
	t.Logf("legacy planned: %d", len(rep.LegacyPlanned))
}

func TestAlignContractSubset(t *testing.T) {
	contract := PathMethods{
		"/api/v1/ci/runs": {"get": {}},
		"/v1/tasks":       {"post": {}},
	}
	swagger := PathMethods{
		"/api/v1/ci/runs": {"get": {}},
		"/api/v1/runs":    {"post": {}},
	}
	rep := AlignContract(contract, swagger, "/api/v1/", "/v1/")
	if len(rep.Missing) != 0 {
		t.Fatalf("missing=%v", rep.Missing)
	}
	if len(rep.LegacyPlanned) != 1 || rep.LegacyPlanned[0] != "/v1/tasks POST" {
		t.Fatalf("legacy=%v", rep.LegacyPlanned)
	}
	if len(rep.Undocumented) != 1 || rep.Undocumented[0] != "/api/v1/runs POST" {
		t.Fatalf("undocumented=%v", rep.Undocumented)
	}
}
