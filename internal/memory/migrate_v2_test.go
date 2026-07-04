package memory

import (
	"testing"
	"time"

	"github.com/ash-repwiki/ash/internal/store"
)

func TestRunMigrations_v1ToV2PreservesExplicitTTL(t *testing.T) {
	svc, _, _ := newTestMemory(t)
	now := time.Now().UTC()
	ttl := 30
	if err := svc.gdb().Create(&store.SchemaMeta{
		Key: MemoryCatalogMetaKey, Value: "1", UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	recordID := "mem_v1_explicit_ttl"
	if err := svc.gdb().Create(&store.MemoryRecord{
		ID: recordID, Layer: "L2", Status: "approved", SpaceID: "local",
		SchemaVersion: 1, Title: "kept", Body: "explicit ttl", TTLDays: &ttl,
		CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	resp, err := svc.RunMigrations(RunMigrationRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if resp.ToVersion != 2 || resp.RecordsUpdated != 1 {
		t.Fatalf("resp=%+v", resp)
	}
	var row store.MemoryRecord
	if err := svc.gdb().First(&row, "id = ?", recordID).Error; err != nil {
		t.Fatal(err)
	}
	if row.SchemaVersion != 2 || row.TTLDays == nil || *row.TTLDays != 30 {
		t.Fatalf("row=%+v want schema 2 and ttl 30", row)
	}
}

func TestDefaultTTLForLayer(t *testing.T) {
	if d := DefaultTTLForLayer("L1"); d == nil || *d != DefaultTTLDaysL1 {
		t.Fatalf("L1=%v", d)
	}
	if d := DefaultTTLForLayer("L2"); d == nil || *d != DefaultTTLDaysL2 {
		t.Fatalf("L2=%v", d)
	}
	if d := DefaultTTLForLayer("L0"); d != nil {
		t.Fatalf("L0=%v want nil", d)
	}
}
