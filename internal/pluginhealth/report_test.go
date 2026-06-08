package pluginhealth

import (
	"testing"
	"time"

	"github.com/ash-repwiki/ash/internal/store"
)

func TestRecordExport_updatesCounters(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	now := time.Now().UTC()
	if err := db.Create(&store.PluginRegistry{
		ID: "plg_health_test", SpaceID: "local", Name: "otel-exporter", Version: "1.0.0",
		Protocol: "grpc", ABI: "ash.plugin.v1", Status: "registered", Compatible: true,
		CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	if err := RecordExport(db.DB, "local", "plg_health_test", false, 2); err != nil {
		t.Fatal(err)
	}
	var row store.PluginRegistry
	if err := db.First(&row, "id = ?", "plg_health_test").Error; err != nil {
		t.Fatal(err)
	}
	if row.ExportErrors != 1 {
		t.Fatalf("exportErrors=%d want 1", row.ExportErrors)
	}
	if row.DropCount != 2 {
		t.Fatalf("dropCount=%d want 2", row.DropCount)
	}
	if row.LastExportAt == nil {
		t.Fatal("expected lastExportAt")
	}

	if err := RecordExport(db.DB, "local", "plg_health_test", true, 0); err != nil {
		t.Fatal(err)
	}
	if err := db.First(&row, "id = ?", "plg_health_test").Error; err != nil {
		t.Fatal(err)
	}
	if row.ExportErrors != 1 {
		t.Fatalf("exportErrors=%d want unchanged 1", row.ExportErrors)
	}
}
