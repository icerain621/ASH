package rag

import (
	"time"

	"github.com/ash-repwiki/ash/internal/store"
	"gorm.io/gorm"
)

// CountChunkFallbackQueries counts run events where RAG fell back to chunk/LIKE search.
func CountChunkFallbackQueries(db *gorm.DB, spaceID string) int64 {
	if db == nil {
		return 0
	}
	runIDs := db.Model(&store.RunRecord{}).Select("id").Where("space_id = ?", spaceID)
	var n int64
	_ = db.Model(&store.RunEvent{}).
		Where("type = ? AND payload_json LIKE ?", "rag.retrieved", `%retrievalMode":"chunk"%`).
		Where("run_id IN (?)", runIDs).
		Count(&n).Error
	return n
}

// FallbackRateInWindow returns chunk-mode RAG queries / total rag.retrieved in the window.
func FallbackRateInWindow(db *gorm.DB, spaceID string, since time.Time) (rate float64, total, chunk int64) {
	if db == nil {
		return 0, 0, 0
	}
	base := db.Model(&store.RunEvent{}).
		Joins("INNER JOIN runs ON runs.id = run_events.run_id").
		Where("runs.space_id = ? AND run_events.type = ? AND run_events.created_at >= ?", spaceID, "rag.retrieved", since)
	_ = base.Count(&total).Error
	if total == 0 {
		return 0, 0, 0
	}
	_ = db.Model(&store.RunEvent{}).
		Joins("INNER JOIN runs ON runs.id = run_events.run_id").
		Where("runs.space_id = ? AND run_events.type = ? AND run_events.created_at >= ?", spaceID, "rag.retrieved", since).
		Where("run_events.payload_json LIKE ?", `%retrievalMode":"chunk"%`).
		Count(&chunk).Error
	if chunk == 0 {
		return 0, total, 0
	}
	return float64(chunk) / float64(total), total, chunk
}
