package store

import (
	"path/filepath"
	"testing"
)

func TestOpenDefaultsToSQLite(t *testing.T) {
	t.Setenv("ASH_DATABASE_URL", "")
	db, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if db.Dialect() != "sqlite" {
		t.Fatalf("dialect=%q want sqlite", db.Dialect())
	}
}

func TestResolveDatabaseTargetProfiles(t *testing.T) {
	dir := t.TempDir()
	tests := []struct {
		name    string
		raw     string
		dialect string
		dsn     string
		wantErr bool
	}{
		{
			name:    "empty dev sqlite",
			raw:     "",
			dialect: "sqlite",
			dsn:     filepath.Join(dir, "ash.db"),
		},
		{
			name:    "sqlite absolute url",
			raw:     "sqlite:///tmp/ash-profile.db",
			dialect: "sqlite",
			dsn:     "/tmp/ash-profile.db",
		},
		{
			name:    "sqlite relative url",
			raw:     "sqlite://profile.db",
			dialect: "sqlite",
			dsn:     filepath.Join(dir, "profile.db"),
		},
		{
			name:    "sqlite file uri",
			raw:     "file::memory:?cache=shared",
			dialect: "sqlite",
			dsn:     "file::memory:?cache=shared",
		},
		{
			name:    "postgres url",
			raw:     "postgres://ash:ash@127.0.0.1:5432/ash?sslmode=disable",
			dialect: "postgres",
			dsn:     "postgres://ash:ash@127.0.0.1:5432/ash?sslmode=disable",
		},
		{
			name:    "postgresql url",
			raw:     "postgresql://ash:ash@127.0.0.1:5432/ash",
			dialect: "postgres",
			dsn:     "postgresql://ash:ash@127.0.0.1:5432/ash",
		},
		{
			name:    "postgres keyword dsn",
			raw:     "host=127.0.0.1 user=ash dbname=ash sslmode=disable",
			dialect: "postgres",
			dsn:     "host=127.0.0.1 user=ash dbname=ash sslmode=disable",
		},
		{
			name:    "unsupported url scheme",
			raw:     "mysql://ash:ash@127.0.0.1/ash",
			wantErr: true,
		},
		{
			name:    "unsupported bare value",
			raw:     "ash-prod",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target, err := resolveDatabaseTarget(dir, tt.raw)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("resolveDatabaseTarget() error=nil want error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if target.dialect != tt.dialect || target.dsn != tt.dsn {
				t.Fatalf("target=%+v want dialect=%q dsn=%q", target, tt.dialect, tt.dsn)
			}
		})
	}
}
