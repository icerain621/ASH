package rag

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func TestCtagsIndexerParsesFixture(t *testing.T) {
	root := repoRoot(t)
	fakeCtags := filepath.Join(root, "scripts", "fixtures", "fake-ctags.sh")
	t.Setenv("CTAGS", fakeCtags)

	dir := t.TempDir()
	src := filepath.Join(dir, "sample.go")
	content := []byte("package main\n\n// comment\n\n\n\nfunc FixtureFunc() {}\n")
	if err := os.WriteFile(src, content, 0o644); err != nil {
		t.Fatal(err)
	}

	idx := NewCtagsIndexer(fakeCtags)
	hits, err := idx.IndexFile(src, content)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 {
		t.Fatalf("hits=%v want 1", hits)
	}
	h := hits[0]
	if h.Name != "FixtureFunc" || h.Kind != "function" || h.Line != 7 {
		t.Fatalf("hit=%+v want FixtureFunc/function/7", h)
	}
	if idx.Name() != "ctags" {
		t.Fatalf("Name()=%q want ctags", idx.Name())
	}
}

func TestRegexIndexerMatchesFunc(t *testing.T) {
	idx := RegexIndexer{}
	content := []byte("package a\n\nfunc Alpha() {}\n")
	hits, err := idx.IndexFile("a.go", content)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || hits[0].Name != "Alpha" || hits[0].Kind != "func" || hits[0].Line != 3 {
		t.Fatalf("hits=%+v", hits)
	}
	if idx.Name() != "regex" {
		t.Fatalf("Name()=%q want regex", idx.Name())
	}
}

func TestResolveSymbolIndexerForcedRegex(t *testing.T) {
	t.Setenv("ASH_RAG_CTAGS", "0")
	t.Setenv("CTAGS", filepath.Join(repoRoot(t), "scripts", "fixtures", "fake-ctags.sh"))
	idx := ResolveSymbolIndexer()
	if idx.Name() != "regex" {
		t.Fatalf("Name()=%q want regex", idx.Name())
	}
}
