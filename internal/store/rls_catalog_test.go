package store

import (
	"strings"
	"testing"
)

func TestMigrationCatalog_RLSCoverage(t *testing.T) {
	if gaps := MigrationCatalogRLSGaps(); len(gaps) > 0 {
		t.Fatalf("migration catalog tables without RLS classification: %v", gaps)
	}
}

func TestVerifyRLSMigrationSQL(t *testing.T) {
	if err := VerifyRLSMigrationSQL(); err != nil {
		t.Fatal(err)
	}
}

func TestPostgresRLSDeferredTables_notClassifiedAsProtected(t *testing.T) {
	protected := postgresRLSClassifiedTables()
	for _, ent := range PostgresRLSDeferredTables() {
		if _, inGlobal := toSet(PostgresRLSGlobalTables())[ent.Table]; inGlobal {
			t.Fatalf("deferred table %q must not be global", ent.Table)
		}
		if _, inTenant := lookupRLSTable(ent.Table); inTenant {
			t.Fatalf("deferred table %q must not be tenant-scoped", ent.Table)
		}
		if containsString(postgresRLSRunScopedTables(), ent.Table) {
			t.Fatalf("deferred table %q must not be run-scoped", ent.Table)
		}
		if containsString(postgresRLSMemoryScopedTables(), ent.Table) {
			t.Fatalf("deferred table %q must not be memory-scoped", ent.Table)
		}
		if _, ok := protected[ent.Table]; !ok {
			t.Fatalf("deferred table %q missing from classified set", ent.Table)
		}
	}
}

func TestFormatRLSCatalogSummary(t *testing.T) {
	s := FormatRLSCatalogSummary()
	if !strings.Contains(s, "policies=41") || !strings.Contains(s, "org=4") {
		t.Fatalf("summary=%q", s)
	}
}

func lookupRLSTable(name string) (PostgresRLSTable, bool) {
	for _, tbl := range PostgresRLSTables() {
		if tbl.Table == name {
			return tbl, true
		}
	}
	return PostgresRLSTable{}, false
}

func containsString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

func toSet(items []string) map[string]struct{} {
	out := make(map[string]struct{}, len(items))
	for _, item := range items {
		out[item] = struct{}{}
	}
	return out
}
