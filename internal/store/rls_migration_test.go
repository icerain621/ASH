package store

import (
	"strings"
	"testing"

	"github.com/ash-repwiki/ash/internal/store/sqlmigrations"
)

func TestRLSMigration_coversCatalog(t *testing.T) {
	if err := VerifyRLSMigrationSQL(); err != nil {
		t.Fatal(err)
	}
	raw, err := sqlmigrations.ReadPostgresUpSQL("000013_rls_tenant_policies.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	dynamicCreates := strings.Count(raw, "'CREATE POLICY %I ON %I")
	if dynamicCreates < 2 {
		t.Fatalf("expected dynamic policy DDL blocks in 000013, got %d", dynamicCreates)
	}
}

func TestRLSExpectedPolicyCount(t *testing.T) {
	if PostgresRLSExpectedPolicyCount() != 41 {
		t.Fatalf("policy count=%d want 41", PostgresRLSExpectedPolicyCount())
	}
	if sqlmigrations.ExpectedVersion() < RLSPoliciesSQLRevision {
		t.Fatalf("expectedVersion=%d want >= %d", sqlmigrations.ExpectedVersion(), RLSPoliciesSQLRevision)
	}
}

func TestRLSGlobalTables_notInTenantCatalog(t *testing.T) {
	protected := make(map[string]struct{})
	for _, tbl := range PostgresRLSTables() {
		protected[tbl.Table] = struct{}{}
	}
	for _, tbl := range PostgresRLSRunScopedTables() {
		protected[tbl] = struct{}{}
	}
	for _, name := range PostgresRLSGlobalTables() {
		if _, ok := protected[name]; ok {
			t.Fatalf("global table %q must not appear in tenant RLS catalog", name)
		}
	}
	raw, err := sqlmigrations.ReadPostgresUpSQL("000013_rls_tenant_policies.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range PostgresRLSGlobalTables() {
		if strings.Contains(raw, "('"+name+"'") {
			t.Fatalf("migration 000013 must not tenant-scope global table %q", name)
		}
	}
}
