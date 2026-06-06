package rag

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ash-repwiki/ash/internal/store"
)

func TestIndexAndQueryUsesFTSWithSymbolRefs(t *testing.T) {
	db, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	svc := NewService(db)
	if err := svc.ensureFTS(); err != nil {
		t.Skipf("sqlite fts5 unavailable: %v", err)
	}

	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, "internal"), 0o755); err != nil {
		t.Fatal(err)
	}
	code := `package internal

func ReconcilePaymentGateway() string {
	return "payment gateway reconciliation reconciliation reconciliation"
}
`
	if err := os.WriteFile(filepath.Join(repo, "internal", "payments.go"), []byte(code), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("payment gateway overview\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	indexed, err := svc.Index(IndexRequest{RepoRoot: repo, SpaceID: "space_rag_test"})
	if err != nil {
		t.Fatal(err)
	}
	if indexed.Documents != 2 || indexed.Chunks != 2 {
		t.Fatalf("indexed=%+v want 2 documents and 2 chunks", indexed)
	}
	var ftsRows int64
	if err := db.Raw("SELECT count(*) FROM rag_chunks_fts").Scan(&ftsRows).Error; err != nil {
		t.Fatal(err)
	}
	if ftsRows != 2 {
		t.Fatalf("ftsRows=%d want 2", ftsRows)
	}

	resp, err := svc.Query(QueryRequest{
		RepoRoot: repo, SpaceID: "space_rag_test", Text: "reconciliation payment gateway", TopK: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Items) == 0 {
		t.Fatal("expected at least one hit")
	}
	top := resp.Items[0]
	if top.Path != "internal/payments.go" {
		t.Fatalf("top path=%q want internal/payments.go: %+v", top.Path, resp.Items)
	}
	if top.Symbol != "ReconcilePaymentGateway" {
		t.Fatalf("symbol=%q want ReconcilePaymentGateway", top.Symbol)
	}
	if !strings.Contains(top.Ref, "internal/payments.go#ReconcilePaymentGateway:") {
		t.Fatalf("ref=%q missing symbol ref", top.Ref)
	}
	if !strings.HasPrefix(top.Digest, "sha256:") {
		t.Fatalf("digest=%q want sha256 prefix", top.Digest)
	}
}

func TestQueryFallsBackToChunkSearchWhenFTSUnavailable(t *testing.T) {
	db, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	svc := NewService(db)
	repo := t.TempDir()
	if err := os.WriteFile(filepath.Join(repo, "note.md"), []byte("fallback evidence works\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Index(IndexRequest{RepoRoot: repo}); err != nil {
		t.Fatal(err)
	}
	if err := svc.ensureFTS(); err == nil {
		if err := db.Exec("DROP TABLE rag_chunks_fts").Error; err != nil {
			t.Fatal(err)
		}
	}

	resp, err := svc.Query(QueryRequest{RepoRoot: repo, Text: "fallback evidence", TopK: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Items) != 1 || resp.Items[0].Path != "note.md" {
		t.Fatalf("hits=%+v want note.md fallback hit", resp.Items)
	}
}

func TestIndexPersistsAbsoluteRepoRoot(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	svc := NewService(db)
	repo := t.TempDir()
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("doctor probe evidence\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Index(IndexRequest{RepoRoot: repo}); err != nil {
		t.Fatal(err)
	}

	absRepo, err := AbsRepoRoot(repo)
	if err != nil {
		t.Fatal(err)
	}
	var chunkCount int64
	if err := db.Model(&store.RAGChunk{}).Where("repo_root = ?", absRepo).Count(&chunkCount).Error; err != nil {
		t.Fatal(err)
	}
	if chunkCount == 0 {
		t.Fatal("expected chunks keyed by absolute repo_root")
	}

	parent := filepath.Dir(repo)
	name := filepath.Base(repo)
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(parent); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(wd) }()

	relAbs, err := AbsRepoRoot(name)
	if err != nil {
		t.Fatal(err)
	}
	if relAbs != absRepo {
		t.Fatalf("relative abs=%q want %q", relAbs, absRepo)
	}
}
