package derive

import (
	"github.com/ash-repwiki/ash/internal/store"
	"gorm.io/gorm"
)

// LoadOptions scopes event replay to one tenant space when SpaceID is set.
type LoadOptions struct {
	SpaceID string
}

// LoadFromDB reads run_events in replay order for offline metric derivation.
func LoadFromDB(db *gorm.DB, opts LoadOptions) ([]Event, error) {
	if db == nil {
		return nil, nil
	}
	q := db.Model(&store.RunEvent{}).
		Select("run_events.run_id, run_events.type, run_events.payload_json").
		Order("run_events.run_id asc, run_events.seq asc")
	if opts.SpaceID != "" {
		q = q.Joins("JOIN runs ON runs.id = run_events.run_id").
			Where("runs.space_id = ?", opts.SpaceID)
	}
	var rows []struct {
		RunID       string
		Type        string
		PayloadJSON string `gorm:"column:payload_json"`
	}
	if err := q.Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]Event, len(rows))
	for i, row := range rows {
		out[i] = Event{
			RunID:       row.RunID,
			Type:        row.Type,
			PayloadJSON: row.PayloadJSON,
		}
	}
	return out, nil
}
