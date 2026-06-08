package rag

import (
	"github.com/ash-repwiki/ash/internal/store"
)

// Profile summarizes retrieval wiring for ops endpoints.
type Profile struct {
	SpaceID              string `json:"spaceId"`
	FtsAvailable         bool   `json:"ftsAvailable"`
	FtsEngine            string `json:"ftsEngine,omitempty"`
	DefaultRetrievalMode string `json:"defaultRetrievalMode"`
	DatabaseDialect      string `json:"databaseDialect"`
	DocumentCount        int64  `json:"documentCount"`
	ChunkCount           int64  `json:"chunkCount"`
	FallbackQueryCount   int64  `json:"fallbackQueryCount"`
}

// Profile returns a tenant-scoped RAG retrieval snapshot.
func (s *Service) Profile(spaceID string) Profile {
	spaceID = firstNonEmpty(spaceID, "local")
	out := Profile{
		SpaceID:              spaceID,
		FtsAvailable:         s.FTSAvailable(),
		FtsEngine:            s.FtsEngine(),
		DefaultRetrievalMode: s.DefaultRetrievalMode(),
	}
	if s.db != nil {
		out.DatabaseDialect = s.db.Dialect()
	}
	if gdb := s.gdb(); gdb != nil {
		_ = gdb.Model(&store.RAGDocument{}).Where("space_id = ?", spaceID).Count(&out.DocumentCount).Error
		_ = gdb.Model(&store.RAGChunk{}).Where("space_id = ?", spaceID).Count(&out.ChunkCount).Error
		out.FallbackQueryCount = CountChunkFallbackQueries(gdb, spaceID)
	}
	return out
}
