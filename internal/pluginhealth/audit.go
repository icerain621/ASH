package pluginhealth

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ash-repwiki/ash/internal/store"
)

// WriteExportAudit records plugin export outcomes in audit_log (appendix D §7).
func WriteExportAudit(db *gorm.DB, spaceID, pluginID, pluginName string, ok bool, dropped int64) error {
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
	eventType := "plugin.export_reported"
	if !ok {
		eventType = "plugin.export_failed"
	}
	payload, err := json.Marshal(map[string]any{
		"pluginId":   pluginID,
		"pluginName": pluginName,
		"ok":         ok,
		"dropped":    dropped,
	})
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	return db.Create(&store.AuditLog{
		ID:          "aud_" + uuid.NewString(),
		SpaceID:     spaceID,
		EventType:   eventType,
		PayloadJSON: string(payload),
		CreatedAt:   now,
	}).Error
}
