package store

import "testing"

func TestRedactDSNPostgresURL(t *testing.T) {
	got := redactDSN("postgres://ash:secret@db.example.com:5432/ash?sslmode=require")
	want := "postgres://ash:***@db.example.com:5432/ash?sslmode=require"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestWorkerConnectionRole(t *testing.T) {
	t.Setenv("ASH_DATABASE_APP_URL", "")
	t.Setenv("ASH_DATABASE_URL", "")
	if WorkerConnectionRole() != "sqlite" {
		t.Fatalf("want sqlite")
	}
	t.Setenv("ASH_DATABASE_URL", "postgres://ash:ash@127.0.0.1:5432/ash?sslmode=disable")
	if WorkerConnectionRole() != "owner" {
		t.Fatalf("want owner got %q", WorkerConnectionRole())
	}
	t.Setenv("ASH_DATABASE_APP_URL", "postgres://ash_app:ash_app@127.0.0.1:5432/ash?sslmode=disable")
	if WorkerConnectionRole() != "ash_app" {
		t.Fatalf("want ash_app got %q", WorkerConnectionRole())
	}
}
