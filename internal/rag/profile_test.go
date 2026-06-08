package rag

import (
	"testing"
	"time"

	"github.com/ash-repwiki/ash/internal/store"
)

func TestProfile_reportsRetrievalMode(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	svc := NewService(db)
	now := time.Now().UTC()
	doc := store.RAGDocument{
		ID: "ragdoc_p", SpaceID: "local", RepoRoot: "/tmp", Path: "a.go",
		Digest: "d", CreatedAt: now, UpdatedAt: now,
	}
	chunk := store.RAGChunk{
		ID: "ragchk_p", DocumentID: doc.ID, SpaceID: "local", RepoRoot: "/tmp", Path: "a.go",
		Text: "x", Digest: "d", CreatedAt: now,
	}
	for _, row := range []any{&doc, &chunk} {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	p := svc.Profile("local")
	if p.DefaultRetrievalMode != RetrievalModeFTS && p.DefaultRetrievalMode != RetrievalModeChunk {
		t.Fatalf("mode=%q", p.DefaultRetrievalMode)
	}
	if p.DocumentCount != 1 || p.ChunkCount != 1 {
		t.Fatalf("counts=%d/%d want 1/1", p.DocumentCount, p.ChunkCount)
	}
}
