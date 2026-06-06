//go:build integration

package store

import (
	"os"
	"testing"
	"time"

	"gorm.io/gorm"
)

func TestPostgresRLSPoliciesInstalled(t *testing.T) {
	pgURL := os.Getenv("ASH_DATABASE_URL")
	if pgURL == "" {
		t.Skip("ASH_DATABASE_URL unset")
	}
	t.Setenv("ASH_POSTGRES_RLS", "1")

	resetPostgresForTest(t, pgURL)
	dir := t.TempDir()
	db, err := OpenWithDatabaseURL(dir, pgURL)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	want := int64(PostgresRLSExpectedPolicyCount())
	count, err := CountPostgresRLSPolicies(db)
	if err != nil {
		t.Fatal(err)
	}
	if count < want {
		t.Fatalf("policies=%d want >= %d", count, want)
	}
}

func TestPostgresRLSSpaceIsolationOnMemoryRecords(t *testing.T) {
	pgURL := os.Getenv("ASH_DATABASE_URL")
	if pgURL == "" {
		t.Skip("ASH_DATABASE_URL unset")
	}
	t.Setenv("ASH_POSTGRES_RLS", "1")
	t.Setenv("ASH_POSTGRES_RLS_FORCE", "1")

	resetPostgresForTest(t, pgURL)
	dir := t.TempDir()
	db, err := OpenWithDatabaseURL(dir, pgURL)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := EnsurePostgresRLSTestRole(db); err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`GRANT ash_rls_tester TO CURRENT_USER`).Error; err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC()
	seed := func(space string) {
		if err := db.TransactionWithRLSBypass(func(tx *gorm.DB) error {
			return tx.Create(&MemoryRecord{
				ID: "mem_" + space, Layer: "team", Status: "approved", SpaceID: space,
				Title: "rls probe", Body: "probe", CreatedAt: now, UpdatedAt: now,
			}).Error
		}); err != nil {
			t.Fatalf("seed %s: %v", space, err)
		}
	}
	seed("space_rls_a")
	seed("space_rls_b")

	var visible []MemoryRecord
	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SET LOCAL ROLE ash_rls_tester").Error; err != nil {
			return err
		}
		if err := SetRLSSession(tx, "space_rls_a", false); err != nil {
			return err
		}
		return tx.Find(&visible).Error
	}); err != nil {
		t.Fatal(err)
	}
	if len(visible) != 1 || visible[0].SpaceID != "space_rls_a" {
		t.Fatalf("visible=%+v want only space_rls_a", visible)
	}

	var leaked int64
	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SET LOCAL ROLE ash_rls_tester").Error; err != nil {
			return err
		}
		if err := SetRLSSession(tx, "space_rls_a", false); err != nil {
			return err
		}
		return tx.Model(&MemoryRecord{}).Where("space_id = ?", "space_rls_b").Count(&leaked).Error
	}); err != nil {
		t.Fatal(err)
	}
	if leaked != 0 {
		t.Fatalf("leaked=%d want 0 cross-space rows under RLS", leaked)
	}
}

