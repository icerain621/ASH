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
	HybridAvailable      bool   `json:"hybridAvailable"`
	DatabaseDialect      string `json:"databaseDialect"`
	DocumentCount        int64  `json:"documentCount"`
	ChunkCount           int64  `json:"chunkCount"`
	PathEntryCount       int64  `json:"pathEntryCount"`
	SymbolCount          int64  `json:"symbolCount"`
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
		_ = gdb.Model(&store.RAGPathEntry{}).Where("space_id = ?", spaceID).Count(&out.PathEntryCount).Error
		_ = gdb.Model(&store.RAGSymbol{}).Where("space_id = ?", spaceID).Count(&out.SymbolCount).Error
		out.FallbackQueryCount = CountChunkFallbackQueries(gdb, spaceID)
	}
	out.HybridAvailable = out.PathEntryCount+out.SymbolCount > 0
	if out.HybridAvailable {
		out.DefaultRetrievalMode = RetrievalModeHybrid
	}
	return out
}
