package api

import "testing"

func TestAssertReadyzScaleParity_match(t *testing.T) {
	if err := AssertReadyzScaleParity(
		HealthResponse{Dialect: "postgres", SQLMigrationExpected: 20, OtelEnabled: true, SchemaMode: "sql", MultiRegion: "disabled"},
		ScaleReadinessResponse{DatabaseDialect: "postgres", SQLMigrationExpected: 20, OtelEnabled: true, SchemaMode: "sql", MultiRegion: "disabled"},
	); err != nil {
		t.Fatal(err)
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
