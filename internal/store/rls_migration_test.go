package store

import (
	"strings"
	"testing"

	"github.com/ash-repwiki/ash/internal/store/sqlmigrations"
)

func TestRLSMigration_coversCatalog(t *testing.T) {
	raw, err := sqlmigrations.ReadPostgresUpSQL("000013_rls_tenant_policies.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, tbl := range PostgresRLSTables() {
		needle := "('" + tbl.Table + "'"
		if !strings.Contains(raw, needle) {
			t.Fatalf("migration missing table %q", tbl.Table)
		}
	}
	for _, tbl := range postgresRLSRunScopedTables() {
		if !strings.Contains(raw, "'"+tbl+"'") {
			t.Fatalf("migration missing run-scoped table %q", tbl)
		}
	}
	policyCount := strings.Count(raw, "CREATE POLICY")
	if policyCount != 0 {
		// policies created via dynamic EXECUTE format inside DO blocks
	}
	dynamicCreates := strings.Count(raw, "'CREATE POLICY %I ON %I")
	if dynamicCreates < 2 {
		t.Fatalf("expected dynamic policy DDL blocks, got %d", dynamicCreates)
	}
}

func TestRLSExpectedPolicyCount(t *testing.T) {
	if PostgresRLSExpectedPolicyCount() != 34 {
		t.Fatalf("policy count=%d want 34", PostgresRLSExpectedPolicyCount())
	}
	if sqlmigrations.ExpectedVersion() < RLSPoliciesSQLRevision {
		t.Fatalf("expectedVersion=%d want >= %d", sqlmigrations.ExpectedVersion(), RLSPoliciesSQLRevision)
	}
}
