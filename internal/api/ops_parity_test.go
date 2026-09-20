package api

import "testing"

func TestAssertReadyzScaleParity_match(t *testing.T) {
	backends := []string{"local", "landlock"}
	if err := AssertReadyzScaleParity(
		HealthResponse{Dialect: "postgres", SQLMigrationExpected: 20, OtelEnabled: true, SchemaMode: "sql", MultiRegion: "disabled", SandboxBackends: backends},
		ScaleReadinessResponse{DatabaseDialect: "postgres", SQLMigrationExpected: 20, OtelEnabled: true, SchemaMode: "sql", MultiRegion: "disabled", SandboxBackends: backends},
	); err != nil {
		t.Fatal(err)
	}
}

func TestAssertReadyzScaleParity_sandboxBackendsMismatch(t *testing.T) {
	if err := AssertReadyzScaleParity(
		HealthResponse{Dialect: "sqlite", SandboxBackends: []string{"local"}},
		ScaleReadinessResponse{DatabaseDialect: "sqlite", SandboxBackends: []string{"docker"}},
	); err == nil {
		t.Fatal("expected sandboxBackends mismatch")
	}
}

func TestAssertReadyzScaleParity_multiRegionMismatch(t *testing.T) {
	if err := AssertReadyzScaleParity(
		HealthResponse{Dialect: "sqlite", MultiRegion: "disabled"},
		ScaleReadinessResponse{DatabaseDialect: "sqlite", MultiRegion: "enabled"},
	); err == nil {
		t.Fatal("expected multiRegion mismatch")
	}
}

func TestAssertReadyzScaleParity_dialectMismatch(t *testing.T) {
	if err := AssertReadyzScaleParity(
		HealthResponse{Dialect: "postgres"},
		ScaleReadinessResponse{DatabaseDialect: "sqlite"},
	); err == nil {
		t.Fatal("expected dialect mismatch")
	}
}
