package rag

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ash-repwiki/ash/internal/store"
)

func buildFakeGopls(t *testing.T) string {
	t.Helper()
	root := repoRoot(t)
	src := filepath.Join(root, "scripts", "fixtures", "fake-gopls")
	out := filepath.Join(t.TempDir(), "fake-gopls")
	if runtime.GOOS == "windows" {
		out += ".exe"
	}
	cmd := exec.Command("go", "build", "-o", out, ".")
	cmd.Dir = src
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build fake-gopls: %v (%s)", err, b)
	}
	return out
}

func TestLSPIndexerDocumentSymbolViaFakeGopls(t *testing.T) {
	bin := buildFakeGopls(t)
	idx := NewLSPIndexer(bin, "")
	if !idx.Available() {
		t.Fatal("Available want true")
	}
	if idx.Name() != "lsp" {
		t.Fatalf("Name=%q", idx.Name())
	}
	dir := t.TempDir()
	src := filepath.Join(dir, "sample.go")
	content := []byte("package main\n\n// pad\n\n\n\nfunc FixtureLSPFunc() {}\n")
	if err := os.WriteFile(src, content, 0o644); err != nil {
		t.Fatal(err)
	}
	hits, err := idx.IndexFile(src, content)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 {
		t.Fatalf("hits=%+v want 1", hits)
	}
	if hits[0].Name != "FixtureLSPFunc" || hits[0].Kind != "func" || hits[0].Line != 7 {
		t.Fatalf("hit=%+v", hits[0])
	}
}

func TestLSPIndexerUnsupportedExt(t *testing.T) {
	idx := NewLSPIndexer("/usr/bin/gopls", "")
	_, err := idx.IndexFile("a.md", []byte("# hi"))
	if err == nil {
		t.Fatal("expected unsupported language error")
	}
}

func TestResolveSymbolIndexerForcedLSP(t *testing.T) {
	bin := buildFakeGopls(t)
	t.Setenv("ASH_RAG_SYMBOL_INDEXER", "lsp")
	t.Setenv(envRAGLSPGopls, bin)
	t.Setenv(envRAGLSPTS, "")
	idx := ResolveSymbolIndexer()
	if idx.Name() != "lsp" {
		t.Fatalf("Name()=%q want lsp", idx.Name())
	}
}

func TestResolveSymbolIndexerLSPViaEnvFlag(t *testing.T) {
	bin := buildFakeGopls(t)
	t.Setenv("ASH_RAG_SYMBOL_INDEXER", "")
	t.Setenv("ASH_RAG_CTAGS", "")
	t.Setenv(envRAGLSP, "1")
	t.Setenv(envRAGLSPGopls, bin)
	idx := ResolveSymbolIndexer()
	if idx.Name() != "lsp" {
		t.Fatalf("Name()=%q want lsp when ASH_RAG_LSP=1", idx.Name())
	}
}

func TestParseDocumentSymbolHierarchical(t *testing.T) {
	raw := []byte(`[{
		"name":"Widget",
		"kind":23,
		"range":{"start":{"line":1,"character":0},"end":{"line":5,"character":1}},
		"selectionRange":{"start":{"line":1,"character":5},"end":{"line":1,"character":11}},
		"children":[{
			"name":"Beta",
			"kind":6,
			"range":{"start":{"line":3,"character":0},"end":{"line":3,"character":20}},
			"selectionRange":{"start":{"line":3,"character":0},"end":{"line":3,"character":4}}
		}]
	}]`)
	hits, err := parseDocumentSymbolResult(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 2 {
		t.Fatalf("hits=%+v", hits)
	}
	if hits[0].Name != "Widget" || hits[0].Kind != "type" || hits[0].Line != 2 {
		t.Fatalf("hits[0]=%+v", hits[0])
	}
	if hits[1].Name != "Beta" || hits[1].Kind != "method" || hits[1].Line != 4 {
		t.Fatalf("hits[1]=%+v", hits[1])
	}
}

func TestRebuildSymbolsLSPWithFakeGopls(t *testing.T) {
	bin := buildFakeGopls(t)
	t.Setenv("ASH_RAG_SYMBOL_INDEXER", "lsp")
	t.Setenv(envRAGLSPGopls, bin)

	db := store.OpenTest(t, t.TempDir())
	svc := NewService(db)
	repo := t.TempDir()
	code := "package p\n\n//\n\n\n\nfunc FixtureLSPFunc() {}\n"
	if err := os.WriteFile(filepath.Join(repo, "lsp.go"), []byte(code), 0o644); err != nil {
		t.Fatal(err)
	}
	resp, err := svc.RebuildSymbols(RebuildSymbolsRequest{RepoRoot: repo, SpaceID: "lsp"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.SymbolSource != "lsp" {
		t.Fatalf("symbolSource=%q want lsp", resp.SymbolSource)
	}
	if resp.Symbols < 1 {
		t.Fatalf("symbols=%d", resp.Symbols)
	}
}
