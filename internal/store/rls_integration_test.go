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
		if err := SetRLSSession(tx, "space_rls_a", "", false); err != nil {
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
		if err := SetRLSSession(tx, "space_rls_a", "", false); err != nil {
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

func TestPostgresRLSSpaceIsolationOnMemoryChildren(t *testing.T) {
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
	seed := func(space, memID string) {
		if err := db.TransactionWithRLSBypass(func(tx *gorm.DB) error {
			if err := tx.Create(&MemoryRecord{
				ID: memID, Layer: "team", Status: "approved", SpaceID: space,
				Title: "rls child probe", Body: "probe", CreatedAt: now, UpdatedAt: now,
			}).Error; err != nil {
				return err
			}
			if err := tx.Create(&MemoryEvidence{
				ID: "ev_" + memID, MemoryID: memID, Kind: "file", Ref: "/probe",
				MetaJSON: "{}", CreatedAt: now,
			}).Error; err != nil {
				return err
			}
			return tx.Create(&MemoryReview{
				ID: "rev_" + memID, MemoryID: memID, Decision: "approve",
				Reason: "rls probe", PolicyProfile: "default", CreatedAt: now,
			}).Error
		}); err != nil {
			t.Fatalf("seed %s: %v", space, err)
		}
	}
	seed("space_rls_a", "mem_child_a")
	seed("space_rls_b", "mem_child_b")

	var evidence []MemoryEvidence
	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SET LOCAL ROLE ash_rls_tester").Error; err != nil {
			return err
		}
		if err := SetRLSSession(tx, "space_rls_a", "", false); err != nil {
			return err
		}
		return tx.Find(&evidence).Error
	}); err != nil {
		t.Fatal(err)
	}
	if len(evidence) != 1 || evidence[0].MemoryID != "mem_child_a" {
		t.Fatalf("evidence=%+v want only mem_child_a", evidence)
	}

	var reviews []MemoryReview
	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SET LOCAL ROLE ash_rls_tester").Error; err != nil {
			return err
		}
		if err := SetRLSSession(tx, "space_rls_a", "", false); err != nil {
			return err
		}
		return tx.Find(&reviews).Error
	}); err != nil {
		t.Fatal(err)
	}
	if len(reviews) != 1 || reviews[0].MemoryID != "mem_child_a" {
		t.Fatalf("reviews=%+v want only mem_child_a", reviews)
	}

	var leakedEvidence int64
	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SET LOCAL ROLE ash_rls_tester").Error; err != nil {
			return err
		}
		if err := SetRLSSession(tx, "space_rls_a", "", false); err != nil {
			return err
		}
		return tx.Model(&MemoryEvidence{}).Where("memory_id = ?", "mem_child_b").Count(&leakedEvidence).Error
	}); err != nil {
		t.Fatal(err)
	}
	if leakedEvidence != 0 {
		t.Fatalf("leaked evidence=%d want 0 cross-space rows under RLS", leakedEvidence)
	}
}

func TestPostgresRLSSpaceIsolationOnOrgIdentity(t *testing.T) {
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
	seed := func(orgID, spaceID, userID, memberID string) {
		if err := db.TransactionWithRLSBypass(func(tx *gorm.DB) error {
			if err := tx.Create(&Org{ID: orgID, Name: orgID, Slug: orgID, CreatedAt: now, UpdatedAt: now}).Error; err != nil {
				return err
			}
			if err := tx.Create(&Space{ID: spaceID, OrgID: orgID, Name: spaceID, Slug: spaceID, CreatedAt: now, UpdatedAt: now}).Error; err != nil {
				return err
			}
			if err := tx.Create(&User{ID: userID, Email: userID + "@example.com", DisplayName: userID, Status: "active", CreatedAt: now, UpdatedAt: now}).Error; err != nil {
				return err
			}
			role := Role{ID: "role_" + orgID, OrgID: orgID, Name: "member", Permissions: "[]", CreatedAt: now, UpdatedAt: now}
			if err := tx.Create(&role).Error; err != nil {
				return err
			}
			return tx.Create(&Member{
				ID: memberID, OrgID: orgID, SpaceID: spaceID, UserID: userID, RoleID: role.ID,
				Status: "active", CreatedAt: now, UpdatedAt: now,
			}).Error
		}); err != nil {
			t.Fatalf("seed %s/%s: %v", orgID, spaceID, err)
		}
	}
	seed("org_rls_a", "space_org_a", "user_a", "mem_a")
	seed("org_rls_b", "space_org_b", "user_b", "mem_b")

	var members []Member
	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SET LOCAL ROLE ash_rls_tester").Error; err != nil {
			return err
		}
		if err := SetRLSSession(tx, "space_org_a", "org_rls_a", false); err != nil {
			return err
		}
		return tx.Find(&members).Error
	}); err != nil {
		t.Fatal(err)
	}
	if len(members) != 1 || members[0].ID != "mem_a" {
		t.Fatalf("members=%+v want only mem_a", members)
	}

	var leaked int64
	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SET LOCAL ROLE ash_rls_tester").Error; err != nil {
			return err
		}
		if err := SetRLSSession(tx, "space_org_a", "org_rls_a", false); err != nil {
			return err
		}
		return tx.Model(&Member{}).Where("org_id = ?", "org_rls_b").Count(&leaked).Error
	}); err != nil {
		t.Fatal(err)
	}
	if leaked != 0 {
		t.Fatalf("leaked members=%d want 0 cross-org rows under RLS", leaked)
	}
}

