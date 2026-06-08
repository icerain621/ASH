package pluginhealth

import (
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/ash-repwiki/ash/internal/store"
)

const (
	OtelExporterPluginID   = "ash-otel-exporter"
	OtelExporterPluginName = "ASH OTel Exporter"
)

// EnsureOtelExporter registers the built-in OTel waterfall exporter plugin row.
func EnsureOtelExporter(db *gorm.DB, spaceID string) error {
	if db == nil {
		return nil
	}
	spaceID = strings.TrimSpace(spaceID)
	if spaceID == "" {
		spaceID = "local"
	}
	now := time.Now().UTC()
	row := store.PluginRegistry{
		ID: OtelExporterPluginID, SpaceID: spaceID, Name: OtelExporterPluginName,
		Version: "0.1.0", Protocol: "internal", ABI: "ash.obs.v0.1",
		Status: "active", Compatible: true, Capabilities: `["otel.trace.export"]`,
		CreatedAt: now, UpdatedAt: now,
	}
	return db.Where("id = ? AND space_id = ?", row.ID, row.SpaceID).
		Assign(map[string]any{
			"name": row.Name, "version": row.Version, "protocol": row.Protocol,
			"abi": row.ABI, "status": row.Status, "compatible": row.Compatible,
			"capabilities": row.Capabilities, "updated_at": now,
		}).FirstOrCreate(&row).Error
}
