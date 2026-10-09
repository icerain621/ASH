package memory

import (
	"strings"

	"github.com/ash-repwiki/ash/internal/store"
)

// KnowledgeHit is a read-only projection of approved knowledge-layer memory.
type KnowledgeHit struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Layer string `json:"layer"`
}

// Knowledge lists approved L2/L3 records for mem.knowledge (read-only; no parallel store).
func (s *Service) Knowledge(spaceID string, limit int) ([]KnowledgeHit, error) {
	spaceID = firstNonEmpty(strings.TrimSpace(spaceID), "local")
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	var rows []store.MemoryRecord
	if err := s.gdb().Where("space_id = ? AND status = ? AND layer IN ?", spaceID, "approved", []string{"L2", "L3"}).
		Order("updated_at desc").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]KnowledgeHit, 0, len(rows))
	for _, row := range rows {
		out = append(out, KnowledgeHit{ID: row.ID, Title: row.Title, Layer: row.Layer})
	}
	return out, nil
}
