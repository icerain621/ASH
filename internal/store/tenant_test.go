package store

import "testing"

func TestEnforceSpaceAccessRejectsCrossTenant(t *testing.T) {
	if err := EnforceSpaceAccess("space_a", "space_b"); err == nil {
		t.Fatal("expected cross-tenant error")
	}
	if err := EnforceSpaceAccess("space_a", "space_a"); err != nil {
		t.Fatalf("same space should pass: %v", err)
	}
}

func TestDatabaseProfileSQLiteDefault(t *testing.T) {
	profile, err := DatabaseProfile(t.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	if profile.Dialect != "sqlite" || !profile.MigrationReady {
		t.Fatalf("profile=%+v", profile)
	}
}

func TestParseDatabaseTargetPostgresURL(t *testing.T) {
	target, err := ParseDatabaseTarget(t.TempDir(), "postgres://ash:ash@127.0.0.1:5432/ash?sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	if target.Dialect != "postgres" {
		t.Fatalf("dialect=%q want postgres", target.Dialect)
	}
}
