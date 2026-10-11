package review

import (
	"encoding/json"
	"strings"

	"github.com/ash-repwiki/ash/internal/memory"
	"github.com/ash-repwiki/ash/internal/store"
)

// LoadMemoryEvalCounts tallies F05 counters from run_events in a space.
func LoadMemoryEvalCounts(db *store.DB, spaceID string, limit int) (EvalCounts, error) {
	kinds, err := ListMemoryFeedbackKinds(db, spaceID, limit)
	if err != nil {
		return EvalCounts{}, err
	}
	return MemoryEvalCounts(kinds), nil
}

// ListMemoryFeedbackKinds extracts feedback kinds (and hit_used events) for a space.
func ListMemoryFeedbackKinds(db *store.DB, spaceID string, limit int) ([]string, error) {
	if db == nil {
		return nil, nil
	}
	spaceID = strings.TrimSpace(spaceID)
	if spaceID == "" {
		spaceID = "local"
	}
	if limit < 1 {
		limit = 5000
	}
	type row struct {
		Type        string
		PayloadJSON string
	}
	var rows []row
	err := db.Table("run_events AS re").
		Select("re.type, re.payload_json").
		Joins("JOIN runs r ON r.id = re.run_id").
		Where("r.space_id = ? AND re.type IN ?", spaceID, []string{"memory.feedback", "memory.hit_used"}).
		Order("re.seq desc").
		Limit(limit).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(rows))
	for _, item := range rows {
		switch item.Type {
		case "memory.hit_used":
			out = append(out, memory.FeedbackHitUsed)
		case "memory.feedback":
			var payload struct {
				Kind string `json:"kind"`
			}
			if err := json.Unmarshal([]byte(item.PayloadJSON), &payload); err != nil {
				continue
			}
			if k := strings.TrimSpace(payload.Kind); k != "" {
				out = append(out, k)
			}
		}
	}
	return out, nil
}
