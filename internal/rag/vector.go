package rag

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/ash-repwiki/ash/internal/store"
)

// embedIndexedFile best-effort embeds chunks for a file into the vector store
// and upserts rag_vector_refs. Failures are swallowed (Index must still succeed).
func (s *Service) embedIndexedFile(space, root, path string) int {
	if s == nil || s.vectors == nil || !s.vectors.Available() {
		return 0
	}
	emb := s.embedder
	if emb == nil {
		emb = DefaultHashEmbedder()
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return 0
	}
	rel = filepath.ToSlash(rel)

	var chunks []store.RAGChunk
	if err := s.gdb().
		Where("space_id = ? AND repo_root = ? AND path = ?", space, root, rel).
		Find(&chunks).Error; err != nil || len(chunks) == 0 {
		return 0
	}

	texts := make([]string, len(chunks))
	for i, c := range chunks {
		texts[i] = c.Text
	}
	vecs, err := emb.Embed(texts)
	if err != nil || len(vecs) != len(chunks) {
		return 0
	}

	col := CollectionForSpace(space)
	points := make([]VectorPoint, len(chunks))
	now := time.Now().UTC()
	refs := make([]store.RAGVectorRef, len(chunks))
	for i, c := range chunks {
		pointID := c.ID
		points[i] = VectorPoint{
			ID:     pointID,
			Vector: vecs[i],
			Payload: map[string]any{
				"chunkId":  c.ID,
				"path":     c.Path,
				"spaceId":  space,
				"repoRoot": root,
			},
		}
		refs[i] = store.RAGVectorRef{
			ID: "ragvec_" + uuid.NewString(), SpaceID: space, RepoRoot: root,
			ChunkID: c.ID, PointID: pointID, CreatedAt: now, UpdatedAt: now,
		}
	}
	if err := s.vectors.Upsert(space, col, points); err != nil {
		return 0
	}

	embedded := 0
	for _, ref := range refs {
		_ = s.gdb().Where("space_id = ? AND repo_root = ? AND chunk_id = ?", space, root, ref.ChunkID).
			Delete(&store.RAGVectorRef{}).Error
		if err := s.gdb().Create(&ref).Error; err != nil {
			continue
		}
		embedded++
	}
	return embedded
}

// queryVectorLane searches the vector store and maps hits to chunk Hits.
// Returns nil when store is down, no refs exist, or search fails (no error to caller).
func (s *Service) queryVectorLane(space, repoRoot, text string, topK int) []Hit {
	if s == nil || s.vectors == nil {
		return nil
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	gdb := s.gdb()
	if gdb == nil {
		return nil
	}
	q := gdb.Model(&store.RAGVectorRef{}).Where("space_id = ?", space)
	if repoRoot != "" {
		q = q.Where("repo_root = ?", repoRoot)
	}
	var refCount int64
	if err := q.Count(&refCount).Error; err != nil || refCount == 0 {
		return nil
	}
	// Probe only when refs exist so normal Query paths stay fast when unused.
	if !s.vectors.Available() {
		return nil
	}

	emb := s.embedder
	if emb == nil {
		emb = DefaultHashEmbedder()
	}
	vecs, err := emb.Embed([]string{text})
	if err != nil || len(vecs) == 0 {
		return nil
	}
	vhits, err := s.vectors.Search(space, CollectionForSpace(space), vecs[0], topK)
	if err != nil || len(vhits) == 0 {
		return nil
	}

	out := make([]Hit, 0, len(vhits))
	for _, vh := range vhits {
		chunkID := payloadString(vh.Payload, "chunkId")
		if chunkID == "" {
			chunkID = vh.ID
		}
		var chunk store.RAGChunk
		cq := gdb.Where("id = ? AND space_id = ?", chunkID, space)
		if repoRoot != "" {
			cq = cq.Where("repo_root = ?", repoRoot)
		}
		if err := cq.First(&chunk).Error; err != nil {
			continue
		}
		score := float64(vh.Score)
		if score == 0 {
			score = 1
		}
		out = append(out, hitFromChunk(chunk, score))
	}
	return out
}

func payloadString(payload map[string]any, key string) string {
	if payload == nil {
		return ""
	}
	v, ok := payload[key]
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	default:
		return ""
	}
}
