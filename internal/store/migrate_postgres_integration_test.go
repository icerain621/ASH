//go:build integration

package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMigratorSQLiteToPostgresE2E(t *testing.T) {
	pgURL := os.Getenv("ASH_DATABASE_URL")
	if pgURL == "" {
		t.Skip("ASH_DATABASE_URL unset")
	}
	if os.Getenv("ASH_MIGRATE_E2E") != "1" {
		t.Skip("set ASH_MIGRATE_E2E=1 for live postgres migrate test")
	}
	resetPostgresForTest(t, pgURL)

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "source.db")

	src, err := OpenSQLite(dir, srcPath)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if err := src.Create(&Org{
		ID: "org_pg_e2e", Name: "PG E2E", Slug: "pg-e2e", CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := src.Create(&RunRecord{
		ID: "run_pg_e2e", TraceID: "trc_pg_e2e", ScenarioName: "feature_delivery",
		ScenarioVersion: "1.0.0", PolicyProfile: "default", Status: "finished",
		SpaceID: "local", StartedAt: now, CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	_ = src.Close()

	m, err := NewMigrator(dir, srcPath, pgURL)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()

	if _, err := m.Copy(CopyOptions{BatchSize: 100}); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Verify(); err != nil {
		t.Fatal(err)
	}
}
