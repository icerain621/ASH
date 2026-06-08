package pluginhealth

import (
	"testing"

	"github.com/ash-repwiki/ash/internal/store"
)

func TestEnsureOtelExporter(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	if err := EnsureOtelExporter(db.DB, "local"); err != nil {
		t.Fatal(err)
	}
	if err := EnsureOtelExporter(db.DB, "local"); err != nil {
		t.Fatal(err)
	}
	var row store.PluginRegistry
	if err := db.First(&row, "id = ? AND space_id = ?", OtelExporterPluginID, "local").Error; err != nil {
		t.Fatal(err)
	}
	if row.Name != OtelExporterPluginName || !row.Compatible {
		t.Fatalf("row=%+v", row)
	}
}

func TestReportExport(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	if err := EnsureOtelExporter(db.DB, "local"); err != nil {
		t.Fatal(err)
	}
	if err := ReportExport(db.DB, "local", OtelExporterPluginID, OtelExporterPluginName, false, 2); err != nil {
		t.Fatal(err)
	}
	var row store.PluginRegistry
	if err := db.First(&row, "id = ?", OtelExporterPluginID).Error; err != nil {
		t.Fatal(err)
	}
	if row.ExportErrors != 1 || row.DropCount != 2 {
		t.Fatalf("exportErrors=%d dropCount=%d", row.ExportErrors, row.DropCount)
	}
	var audits int64
	if err := db.Model(&store.AuditLog{}).Where("event_type = ?", "plugin.export_failed").Count(&audits).Error; err != nil {
		t.Fatal(err)
	}
	if audits != 1 {
		t.Fatalf("audit count=%d want 1", audits)
	}
}
