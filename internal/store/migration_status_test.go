package store

import "testing"

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
