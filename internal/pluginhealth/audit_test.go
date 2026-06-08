package pluginhealth

import (
	"testing"

	"github.com/ash-repwiki/ash/internal/store"
)

func TestWriteExportAudit_failed(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	if err := WriteExportAudit(db.DB, "local", "plg_x", "exporter", false, 1); err != nil {
		t.Fatal(err)
	}
	var row store.AuditLog
	if err := db.Where("event_type = ?", "plugin.export_failed").First(&row).Error; err != nil {
		t.Fatal(err)
	}
	if row.SpaceID != "local" {
		t.Fatalf("space=%q want local", row.SpaceID)
	}
}
