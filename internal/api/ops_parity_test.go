package api

import "testing"

func TestAssertReadyzScaleParity_match(t *testing.T) {
	if err := AssertReadyzScaleParity(
		HealthResponse{Dialect: "postgres", SQLMigrationExpected: 20, OtelEnabled: true, SchemaMode: "sql"},
		ScaleReadinessResponse{DatabaseDialect: "postgres", SQLMigrationExpected: 20, OtelEnabled: true, SchemaMode: "sql"},
	); err != nil {
		t.Fatal(err)
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
