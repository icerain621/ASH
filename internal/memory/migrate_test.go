package memory

import (
	"testing"
	"time"

	"github.com/ash-repwiki/ash/internal/store"
)

func TestRunMigrations_v0ToV1Backfill(t *testing.T) {
	svc, ev := newTestMemory(t)
	now := time.Now().UTC()
	runID := "run_mem_migrate_test"
	traceID := "trace_mem_migrate_test"
	seedRun(t, svc.db, runID, traceID)
	// seedRun sets Status running; migration only needs the row to exist for SSE emit.
	if err := svc.db.Model(&store.RunRecord{}).Where("id = ?", runID).Update("status", "completed").Error; err != nil {
		t.Fatal(err)
	}

	legacyID := "mem_legacy_v0"
	if err := svc.gdb().Create(&store.MemoryRecord{
		ID: legacyID, Layer: "L1", Status: "approved", SpaceID: "local",
		SchemaVersion: legacySchemaVersion, Title: "legacy", Body: "needs backfill",
		CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	resp, err := svc.RunMigrations(RunMigrationRequest{RunID: runID, DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if resp.RecordsUpdated != 2 {
		t.Fatalf("dry-run updated=%d want 2 (v0→v1 + v1→v2)", resp.RecordsUpdated)
	}

	resp, err = svc.RunMigrations(RunMigrationRequest{RunID: runID})
	if err != nil {
		t.Fatal(err)
	}
	if resp.RecordsUpdated != 2 || resp.ToVersion != 2 {
		t.Fatalf("resp=%+v want v2 migration", resp)
	}
	catalog, err := CatalogVersion(svc.db)
	if err != nil || catalog != 2 {
		t.Fatalf("catalog=%d err=%v want 2", catalog, err)
	}
	var row store.MemoryRecord
	if err := svc.gdb().First(&row, "id = ?", legacyID).Error; err != nil {
		t.Fatal(err)
	}
	if row.SchemaVersion != CurrentSchemaVersion || row.DedupeKey == "" {
		t.Fatalf("row=%+v", row)
	}
	if row.TTLDays == nil || *row.TTLDays != DefaultTTLDaysL1 {
		t.Fatalf("ttl=%v want L1 default %d", row.TTLDays, DefaultTTLDaysL1)
	}

	envelopes, err := ev.ListAfter(runID, 0, 200)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, e := range envelopes {
		if e.Type == "memory.migrated" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected memory.migrated run event")
	}

	resp, err = svc.RunMigrations(RunMigrationRequest{RunID: runID})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.AlreadyCurrent {
		t.Fatalf("second run=%+v want alreadyCurrent", resp)
	}
}
