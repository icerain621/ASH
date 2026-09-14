package scoring

import (
	"encoding/json"
	"strings"

	"github.com/ash-repwiki/ash/internal/store"
	"gorm.io/gorm"
)

const eventScoreVoided = "score.voided"

// IsVoided reports whether audit_log contains score.voided for the given score event.
// Void is audit-only: score_events rows are never mutated on appeal reject.
func IsVoided(db *gorm.DB, scoreEventID string) bool {
	scoreEventID = strings.TrimSpace(scoreEventID)
	if db == nil || scoreEventID == "" {
		return false
	}
	var n int64
	_ = db.Model(&store.AuditLog{}).
		Where("event_type = ? AND payload_json LIKE ?", eventScoreVoided, "%\"scoreEventId\":\""+scoreEventID+"\"%").
		Count(&n).Error
	return n > 0
}

// VoidedIDs returns score_event IDs marked score.voided in the space (batch).
func VoidedIDs(db *gorm.DB, spaceID string) (map[string]struct{}, error) {
	out := map[string]struct{}{}
	if db == nil {
		return out, nil
	}
	spaceID = strings.TrimSpace(spaceID)
	if spaceID == "" {
		spaceID = "local"
	}
	var rows []store.AuditLog
	if err := db.Where("space_id = ? AND event_type = ?", spaceID, eventScoreVoided).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		if id := scoreEventIDFromPayload(row.PayloadJSON); id != "" {
			out[id] = struct{}{}
		}
	}
	return out, nil
}

func scoreEventIDFromPayload(payloadJSON string) string {
	payload := map[string]any{}
	if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
		return ""
	}
	if v, ok := payload["scoreEventId"].(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}
