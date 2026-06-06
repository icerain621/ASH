package store

import "testing"

func TestShouldApplyPostgresRLSPolicies(t *testing.T) {
	owner := "postgres://ash:ash@127.0.0.1:5433/ash?sslmode=disable"
	app := "postgres://ash_app:ash_app@127.0.0.1:5433/ash?sslmode=disable"
	t.Setenv("ASH_DATABASE_APP_URL", app)
	if !shouldApplyPostgresRLSPolicies(owner) {
		t.Fatal("owner URL should apply policies")
	}
	if shouldApplyPostgresRLSPolicies(app) {
		t.Fatal("app URL must not apply policies")
	}
	t.Setenv("ASH_DATABASE_APP_URL", "")
	if !shouldApplyPostgresRLSPolicies(owner) {
		t.Fatal("owner URL should apply when app URL unset")
	}
}

func TestPostgresAppDatabaseURL(t *testing.T) {
	got := PostgresAppDatabaseURL("127.0.0.1", 5433, "ash")
	want := "postgres://ash_app:ash_app@127.0.0.1:5433/ash?sslmode=disable"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
