package store

import (
	"testing"
)

func TestRuntimeDatabaseURLPrefersAppURL(t *testing.T) {
	t.Setenv("ASH_DATABASE_URL", "postgres://ash:ash@localhost/ash")
	t.Setenv("ASH_DATABASE_APP_URL", "postgres://ash_app:ash_app@localhost/ash")
	if got := RuntimeDatabaseURL(); got != "postgres://ash_app:ash_app@localhost/ash" {
		t.Fatalf("got %q", got)
	}
}

func TestRuntimeDatabaseURLFallsBackToDatabaseURL(t *testing.T) {
	t.Setenv("ASH_DATABASE_URL", "postgres://ash:ash@localhost/ash")
	t.Setenv("ASH_DATABASE_APP_URL", "")
	if got := RuntimeDatabaseURL(); got != "postgres://ash:ash@localhost/ash" {
		t.Fatalf("got %q", got)
	}
}

func TestPostgresMigrationDSNPrefersOpenDSNOverAppURL(t *testing.T) {
	t.Setenv("ASH_DATABASE_URL", "postgres://ash:ash@localhost/ash")
	t.Setenv("ASH_DATABASE_APP_URL", "postgres://ash_app:ash_app@localhost/ash")
	owner := "postgres://ash:ash@localhost/ash?sslmode=disable"
	db := &DB{dsn: owner}
	if got := postgresMigrationDSN(db); got != owner {
		t.Fatalf("postgresMigrationDSN=%q want open-time owner DSN %q (not ASH_DATABASE_APP_URL)", got, owner)
	}
}

func TestPostgresMigrationDSNFallsBackToRuntimeURL(t *testing.T) {
	t.Setenv("ASH_DATABASE_URL", "postgres://ash:ash@localhost/ash")
	t.Setenv("ASH_DATABASE_APP_URL", "postgres://ash_app:ash_app@localhost/ash")
	db := &DB{dsn: ""}
	want := "postgres://ash_app:ash_app@localhost/ash"
	if got := postgresMigrationDSN(db); got != want {
		t.Fatalf("postgresMigrationDSN=%q want runtime fallback %q", got, want)
	}
}
