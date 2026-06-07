//go:build integration

package store

import (
	"os"
	"testing"

	"github.com/ash-repwiki/ash/internal/store/sqlmigrations"
)

// TestPostgresSQLSchemaModeE2E opens a fresh Postgres with ASH_SCHEMA_MODE=sql (no AutoMigrate).
func TestPostgresSQLSchemaModeE2E(t *testing.T) {
	pgURL := os.Getenv("ASH_DATABASE_URL")
	if pgURL == "" {
		t.Skip("ASH_DATABASE_URL unset")
	}
	if os.Getenv("ASH_MIGRATE_E2E") != "1" {
		t.Skip("set ASH_MIGRATE_E2E=1 for live postgres sql-schema test")
	}
	t.Setenv("ASH_SCHEMA_MODE", "sql")
	resetPostgresForTest(t, pgURL)

	dir := t.TempDir()
	db, err := OpenWithDatabaseURL(dir, pgURL)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if !sqlmigrations.AutoMigrateEnabled("postgres") {
		// expected in sql mode
	} else {
		t.Fatal("automigrate should be disabled when ASH_SCHEMA_MODE=sql")
	}
	if err := VerifyMigrationSchema(db); err != nil {
		t.Fatal(err)
	}
	v, dirty, err := sqlmigrations.VersionPostgres(pgURL)
	if err != nil {
		t.Fatal(err)
	}
	if dirty {
		t.Fatal("schema_migrations dirty")
	}
	want := sqlmigrations.ExpectedVersion()
	if v != want {
		t.Fatalf("sql version=%d want %d", v, want)
	}
}
