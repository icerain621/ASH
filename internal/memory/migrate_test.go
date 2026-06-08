package memory

import (
	"testing"
	"time"

	"github.com/ash-repwiki/ash/internal/store"
)

func TestRunMigrations_v0ToV1Backfill(t *testing.T) {
	svc, ev, runsSvc := newTestMemory(t)
	now := time.Now().UTC()
	runID := "run_mem_migrate_test"
	traceID := "trace_mem_migrate_test"
	if err := runsSvc.DB().Create(&store.RunRecord{
		ID: runID, TraceID: traceID, ScenarioName: "feature_delivery", ScenarioVersion: "1.0.0",
		PolicyProfile: "default", Status: "completed", SpaceID: "local",
		StartedAt: now, CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
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
	if resp.RecordsUpdated != 1 {
		t.Fatalf("dry-run updated=%d want 1", resp.RecordsUpdated)
	}

	resp, err = svc.RunMigrations(RunMigrationRequest{RunID: runID})
	if err != nil {
		t.Fatal(err)
	}
	if resp.RecordsUpdated != 1 || resp.ToVersion != 1 {
		t.Fatalf("resp=%+v", resp)
	}
	catalog, err := CatalogVersion(svc.db)
	if err != nil || catalog != 1 {
		t.Fatalf("catalog=%d err=%v", catalog, err)
	}
	var row store.MemoryRecord
	if err := svc.gdb().First(&row, "id = ?", legacyID).Error; err != nil {
		t.Fatal(err)
	}
	if row.SchemaVersion != CurrentSchemaVersion || row.DedupeKey == "" {
		t.Fatalf("row=%+v", row)
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
