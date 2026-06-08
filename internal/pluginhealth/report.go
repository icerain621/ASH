package pluginhealth

import (
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/ash-repwiki/ash/internal/store"
)

// RecordExport updates plugin export health counters (appendix D §7).
func RecordExport(db *gorm.DB, spaceID, pluginID string, ok bool, dropped int64) error {
	if db == nil {
		return fmt.Errorf("database is nil")
	}
	pluginID = strings.TrimSpace(pluginID)
	if pluginID == "" {
		return fmt.Errorf("pluginId is required")
	}
	spaceID = strings.TrimSpace(spaceID)
	if spaceID == "" {
		spaceID = "local"
	}

	var row store.PluginRegistry
	if err := db.First(&row, "id = ? AND space_id = ?", pluginID, spaceID).Error; err != nil {
		return err
	}
	now := time.Now().UTC()
	patch := map[string]any{
		"last_export_at": now,
		"updated_at":     now,
	}
	if !ok {
		patch["export_errors"] = gorm.Expr("export_errors + ?", 1)
	}
	if dropped > 0 {
		patch["drop_count"] = gorm.Expr("drop_count + ?", dropped)
	}
	return db.Model(&store.PluginRegistry{}).Where("id = ?", pluginID).Updates(patch).Error
}
