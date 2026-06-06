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
