package store

import (
	"testing"
	"time"
)

func TestMigrationCatalogAndSchema(t *testing.T) {
	if MigrationCatalogSize() < 25 {
		t.Fatalf("catalog size=%d want >=25", MigrationCatalogSize())
	}
	db := OpenTest(t, t.TempDir())
	if err := VerifyMigrationSchema(db); err != nil {
		t.Fatal(err)
	}
	snap, err := MigrationSnapshotFor(db.DataDir())
	if err != nil {
		t.Fatal(err)
	}
	if snap.MigrationTableCount != MigrationCatalogSize() {
		t.Fatalf("snapshot count=%d catalog=%d", snap.MigrationTableCount, MigrationCatalogSize())
	}
}

func TestMigrationSnapshotDualWriteHint(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ASH_DUAL_WRITE_POSTGRES_URL", "postgres://ash:secret@shadow.example.com:5432/ash?sslmode=require")
	snap, err := MigrationSnapshotFor(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !snap.DualWriteRuntime {
		t.Fatal("expected dual-write runtime")
	}
	if snap.DualWriteShadowURLHint != "postgres://ash:***@shadow.example.com:5432/ash?sslmode=require" {
		t.Fatalf("hint=%q", snap.DualWriteShadowURLHint)
	}
}

func TestMigrationSnapshotSyncError(t *testing.T) {
	dir := t.TempDir()
	errAt := time.Now().UTC().Add(-time.Minute)
	if err := SaveSyncState(dir, &SyncState{
		LastError:   "copy orgs: database is locked",
		LastErrorAt: &errAt,
		UpdatedAt:   errAt,
	}); err != nil {
		t.Fatal(err)
	}
	snap, err := MigrationSnapshotFor(dir)
	if err != nil {
		t.Fatal(err)
	}
	if snap.LastSyncError != "copy orgs: database is locked" {
		t.Fatalf("lastSyncError=%q", snap.LastSyncError)
	}
	if snap.LastSyncErrorAt == nil || !snap.LastSyncErrorAt.Equal(errAt) {
		t.Fatalf("lastSyncErrorAt=%v want %v", snap.LastSyncErrorAt, errAt)
	}
}
