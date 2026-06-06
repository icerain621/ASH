//go:build integration

package store

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"gorm.io/gorm"
)

// TestPostgresRLSE2EAfterMigrate applies RLS on an already-migrated Postgres (ASH_MIGRATE_E2E=1).
func TestPostgresRLSE2EAfterMigrate(t *testing.T) {
	pgURL := os.Getenv("ASH_DATABASE_URL")
	if pgURL == "" {
		t.Skip("ASH_DATABASE_URL unset")
	}
	if os.Getenv("ASH_MIGRATE_E2E") != "1" {
		t.Skip("ASH_MIGRATE_E2E unset")
	}
	t.Setenv("ASH_POSTGRES_RLS", "1")
	t.Setenv("ASH_POSTGRES_RLS_FORCE", "1")

	dir := t.TempDir()
	admin, err := OpenWithDatabaseURL(dir, pgURL)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	if err := EnsurePostgresAppRole(admin); err != nil {
		t.Fatal(err)
	}
	want := int64(PostgresRLSExpectedPolicyCount())
	count, err := CountPostgresRLSPolicies(admin)
	if err != nil {
		t.Fatal(err)
	}
	if count < want {
		t.Fatalf("policies=%d want >= %d (run with ASH_POSTGRES_RLS=1 on migrated db)", count, want)
	}

	appURL := os.Getenv("ASH_DATABASE_APP_URL")
	if appURL == "" {
		appURL = PostgresAppDatabaseURL("127.0.0.1", 5432, "ash")
	}
	t.Setenv("ASH_DATABASE_APP_URL", appURL)

	appDir := t.TempDir()
	appDB, err := Open(appDir)
	if err != nil {
		t.Fatal(err)
	}
	defer appDB.Close()
	if appDB.Dialect() != "postgres" {
		t.Fatalf("dialect=%q want postgres", appDB.Dialect())
	}
	sqlDB, err := appDB.DB.DB()
	if err != nil {
		t.Fatal(err)
	}
	if err := pingDB(sqlDB); err != nil {
		t.Fatal(err)
	}

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	spaceA := "space_rls_e2e_a_" + suffix
	spaceB := "space_rls_e2e_b_" + suffix
	now := time.Now().UTC()
	if err := admin.TransactionWithRLSBypass(func(tx *gorm.DB) error {
		return tx.Create(&MemoryRecord{
			ID: "mem_rls_e2e_" + suffix, Layer: "team", Status: "approved", SpaceID: spaceA,
			Title: "e2e", Body: "e2e", CreatedAt: now, UpdatedAt: now,
		}).Error
	}); err != nil {
		t.Fatal(err)
	}

	ctx := WithRLSSpaceContext(context.Background(), spaceA)
	var visible int64
	if err := appDB.WithContext(ctx).Model(&MemoryRecord{}).Where("space_id = ?", spaceB).Count(&visible).Error; err != nil {
		t.Fatal(err)
	}
	if visible != 0 {
		t.Fatalf("ash_app leaked %d rows for foreign space", visible)
	}
}

func pingDB(db *sql.DB) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return db.PingContext(ctx)
}
