package sqlmigrations

import (
	"io/fs"
	"os"
	"strings"
	"testing"
)

func TestMode_defaultsDual(t *testing.T) {
	t.Setenv(envSchemaMode, "")
	t.Setenv(envDisableAuto, "")
	if Mode() != SchemaModeDual {
		t.Fatalf("mode=%q want dual", Mode())
	}
}

func TestMode_disableAutomigrateForcesSQL(t *testing.T) {
	t.Setenv(envDisableAuto, "1")
	if Mode() != SchemaModeSQL {
		t.Fatalf("mode=%q want sql", Mode())
	}
}

func TestSQLMigrationsEnabled_postgresOnly(t *testing.T) {
	t.Setenv(envSchemaMode, SchemaModeDual)
	if !SQLMigrationsEnabled("postgres") {
		t.Fatal("expected postgres sql migrations enabled in dual mode")
	}
	if SQLMigrationsEnabled("sqlite") {
		t.Fatal("sqlite should not use golang-migrate yet")
	}
}

func TestAutoMigrateEnabled_modes(t *testing.T) {
	t.Setenv(envSchemaMode, SchemaModeSQL)
	if AutoMigrateEnabled("postgres") {
		t.Fatal("sql mode should disable automigrate")
	}
	t.Setenv(envSchemaMode, SchemaModeAuto)
	if !AutoMigrateEnabled("postgres") {
		t.Fatal("automigrate mode should allow automigrate")
	}
}

func TestExpectedVersion_matchesEmbeddedUpFiles(t *testing.T) {
	var count int
	err := fs.WalkDir(postgresFiles, "migrations/postgres", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".up.sql") {
			count++
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if uint(count) != ExpectedVersion() {
		t.Fatalf("embedded up migrations=%d expectedVersion=%d", count, ExpectedVersion())
	}
}

func TestApplyPostgres_requiresDSN(t *testing.T) {
	if os.Getenv("ASH_MIGRATE_E2E") != "1" {
		t.Skip("set ASH_MIGRATE_E2E=1 with live Postgres to run")
	}
	dsn := os.Getenv("ASH_DATABASE_URL")
	if dsn == "" {
		t.Skip("ASH_DATABASE_URL required")
	}
	v, err := ApplyPostgres(dsn)
	if err != nil {
		t.Fatal(err)
	}
	if v < 1 {
		t.Fatalf("version=%d", v)
	}
}
