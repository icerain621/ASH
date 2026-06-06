package store

import (
	"strings"
	"testing"
)

func TestPostgresRLSPolicyExprIncludesBypassAndSpace(t *testing.T) {
	expr := postgresRLSPolicyExpr("space_id")
	if !strings.Contains(expr, RLSBypassSetting) || !strings.Contains(expr, RLSSpaceSetting) {
		t.Fatalf("expr=%q missing bypass/space settings", expr)
	}
	if !strings.Contains(expr, `"space_id"`) {
		t.Fatalf("expr=%q missing space_id column", expr)
	}
}

func TestPostgresRLSTablesCoverCoreTenantData(t *testing.T) {
	names := map[string]struct{}{}
	for _, tbl := range PostgresRLSTables() {
		names[tbl.Table] = struct{}{}
	}
	for _, required := range []string{"runs", "memory_records", "secret_records", "audit_log", "spaces"} {
		if _, ok := names[required]; !ok {
			t.Fatalf("missing RLS table %q", required)
		}
	}
}

func TestApplyPostgresRLSNoOpOnSQLite(t *testing.T) {
	db := OpenTest(t, t.TempDir())
	if err := ApplyPostgresRLSPolicies(db); err != nil {
		t.Fatal(err)
	}
	count, err := CountPostgresRLSPolicies(db)
	if err != nil || count != 0 {
		t.Fatalf("count=%d err=%v want 0 on sqlite", count, err)
	}
}
