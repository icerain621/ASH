package store

import (
	"testing"

	"github.com/ash-repwiki/ash/internal/store/sqlmigrations"
)

func TestSchemaProfile_sqliteDefaults(t *testing.T) {
	t.Setenv("ASH_SCHEMA_MODE", "")
	info := SchemaProfile("sqlite", "")
	if info.Mode != sqlmigrations.SchemaModeDual {
		t.Fatalf("mode=%q", info.Mode)
	}
	if info.SQLMigrationsEnabled {
		t.Fatal("sqlite should not enable sql migrations")
	}
	if !info.AutoMigrateEnabled {
		t.Fatal("sqlite should use automigrate")
	}
	if info.SQLMigrationExpected != sqlmigrations.ExpectedVersion() {
		t.Fatalf("expected=%d", info.SQLMigrationExpected)
	}
}

func TestSchemaProfile_postgresSQLMode(t *testing.T) {
	t.Setenv("ASH_SCHEMA_MODE", "sql")
	info := SchemaProfile("postgres", "")
	if info.Mode != sqlmigrations.SchemaModeSQL {
		t.Fatalf("mode=%q", info.Mode)
	}
	if !info.SQLMigrationsEnabled {
		t.Fatal("postgres sql mode should enable sql migrations")
	}
	if info.AutoMigrateEnabled {
		t.Fatal("sql mode should disable automigrate")
	}
}
