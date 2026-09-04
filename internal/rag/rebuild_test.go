package rag

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ash-repwiki/ash/internal/store"
)

func TestRebuildSymbolsUpsertsAndCleansStale(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	svc := NewService(db)
	repo := t.TempDir()
	mustWrite := func(rel, body string) {
		p := filepath.Join(repo, rel)
		_ = os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mustWrite("a.go", "package a\n\nfunc Alpha() {}\n")
	mustWrite("b.go", "package b\n\nfunc Beta() {}\n")
	resp, err := svc.RebuildSymbols(RebuildSymbolsRequest{RepoRoot: repo, SpaceID: "s1"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Paths < 2 || resp.Symbols < 2 {
		t.Fatalf("resp=%+v", resp)
	}
	if resp.SymbolSource != "regex" {
		t.Fatalf("SymbolSource=%q want regex", resp.SymbolSource)
	}
	var sym store.RAGSymbol
	if err := db.Where("space_id = ? AND name = ?", "s1", "Alpha").First(&sym).Error; err != nil {
		t.Fatal(err)
	}
	if sym.Source != "regex" {
		t.Fatalf("sym.Source=%q want regex", sym.Source)
	}
	_ = os.Remove(filepath.Join(repo, "b.go"))
	resp2, err := svc.RebuildSymbols(RebuildSymbolsRequest{RepoRoot: repo, SpaceID: "s1"})
	if err != nil {
		t.Fatal(err)
	}
	var paths, syms int64
	_ = db.Model(&store.RAGPathEntry{}).Where("space_id = ?", "s1").Count(&paths)
	_ = db.Model(&store.RAGSymbol{}).Where("space_id = ?", "s1").Count(&syms)
	if paths != 1 {
		t.Fatalf("paths=%d want 1 after stale clean", paths)
	}
	if syms < 1 {
		t.Fatalf("syms=%d", syms)
	}
	_ = resp2
}
