package rag

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ash-repwiki/ash/internal/store"
)

func TestHybridQueryPrefersSymbolOverNoise(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	svc := NewService(db)
	repo := t.TempDir()
	_ = os.WriteFile(filepath.Join(repo, "noise.md"), []byte(strings.Repeat("payment ", 40)+"\n"), 0o644)
	_ = os.WriteFile(filepath.Join(repo, "pay.go"), []byte("package p\n\nfunc UniqueHybridSymbol() {}\n"), 0o644)
	if _, err := svc.Index(IndexRequest{RepoRoot: repo, SpaceID: "hy"}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.RebuildSymbols(RebuildSymbolsRequest{RepoRoot: repo, SpaceID: "hy"}); err != nil {
		t.Fatal(err)
	}
	resp, err := svc.Query(QueryRequest{RepoRoot: repo, SpaceID: "hy", Text: "UniqueHybridSymbol", TopK: 3, Prefer: "symbol"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.RetrievalMode != RetrievalModeHybrid {
		t.Fatalf("mode=%s", resp.RetrievalMode)
	}
	if len(resp.Items) == 0 || resp.Items[0].Symbol != "UniqueHybridSymbol" {
		t.Fatalf("items=%+v", resp.Items)
	}
}

func TestQueryFallsBackWhenHybridEmpty(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	svc := NewService(db)
	repo := t.TempDir()
	_ = os.WriteFile(filepath.Join(repo, "note.md"), []byte("fallback only text\n"), 0o644)
	if _, err := svc.Index(IndexRequest{RepoRoot: repo, SpaceID: "fb"}); err != nil {
		t.Fatal(err)
	}
	resp, err := svc.Query(QueryRequest{RepoRoot: repo, SpaceID: "fb", Text: "fallback", TopK: 3})
	if err != nil {
		t.Fatal(err)
	}
	if resp.RetrievalMode == RetrievalModeHybrid {
		t.Fatalf("mode=%s want fts or chunk when hybrid tables empty", resp.RetrievalMode)
	}
	if resp.RetrievalMode != RetrievalModeFTS && resp.RetrievalMode != RetrievalModeChunk {
		t.Fatalf("mode=%s want fts or chunk", resp.RetrievalMode)
	}
	if len(resp.Items) == 0 {
		t.Fatal("expected at least one hit")
	}
}

func TestQueryInvalidPreferReturnsError(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	svc := NewService(db)
	_, err := svc.Query(QueryRequest{Text: "hello", Prefer: "bogus"})
	if err == nil {
		t.Fatal("expected error for invalid prefer")
	}
	if !errors.Is(err, ErrInvalidPrefer) {
		t.Fatalf("err=%v want ErrInvalidPrefer", err)
	}
}
