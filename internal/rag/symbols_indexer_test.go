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
	t.Setenv("ASH_RAG_SYMBOL_INDEXER", "regex")
	t.Setenv("ASH_RAG_CTAGS", "0")
	t.Setenv("CTAGS", filepath.Join(repoRoot(t), "scripts", "fixtures", "fake-ctags.sh"))
	idx := ResolveSymbolIndexer()
	if idx.Name() != "regex" {
		t.Fatalf("Name()=%q want regex", idx.Name())
	}
}

func TestResolveSymbolIndexerDefaultTreesitter(t *testing.T) {
	t.Setenv("ASH_RAG_SYMBOL_INDEXER", "")
	t.Setenv("ASH_RAG_CTAGS", "")
	t.Setenv("CTAGS", "")
	idx := ResolveSymbolIndexer()
	if idx.Name() != "treesitter" {
		t.Fatalf("Name()=%q want treesitter", idx.Name())
	}
}

func TestResolveSymbolIndexerForcedTreesitter(t *testing.T) {
	t.Setenv("ASH_RAG_SYMBOL_INDEXER", "treesitter")
	t.Setenv("ASH_RAG_CTAGS", "0")
	idx := ResolveSymbolIndexer()
	if idx.Name() != "treesitter" {
		t.Fatalf("Name()=%q want treesitter", idx.Name())
	}
}

func TestTreeSitterIndexerGoAST(t *testing.T) {
	idx := NewTreeSitterIndexer()
	content := []byte(`package sample

type Widget struct{}

func Alpha() {}

func (Widget) Beta() {}

const Gamma = 1
`)
	hits, err := idx.IndexFile("sample.go", content)
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]SymbolHit{}
	for _, h := range hits {
		byName[h.Name] = h
	}
	for _, want := range []struct {
		name, kind string
	}{
		{"Widget", "type"},
		{"Alpha", "func"},
		{"Beta", "method"},
		{"Gamma", "const"},
	} {
		h, ok := byName[want.name]
		if !ok {
			t.Fatalf("missing %s in %+v", want.name, hits)
		}
		if h.Kind != want.kind {
			t.Fatalf("%s kind=%q want %q", want.name, h.Kind, want.kind)
		}
	}
	if idx.Name() != "treesitter" {
		t.Fatalf("Name()=%q", idx.Name())
	}
}

func TestTreeSitterIndexerJSTS(t *testing.T) {
	idx := NewTreeSitterIndexer()
	content := []byte(`export function greet() {}
export class Box {
  open() {}
}
export type Id = string;
`)
	hits, err := idx.IndexFile("a.ts", content)
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]string{}
	for _, h := range hits {
		byName[h.Name] = h.Kind
	}
	if byName["greet"] != "func" || byName["Box"] != "class" || byName["Id"] != "type" {
		t.Fatalf("hits=%+v", hits)
	}
	if byName["open"] != "method" {
		t.Fatalf("open kind=%q want method; hits=%+v", byName["open"], hits)
	}
}

func TestTreeSitterIndexerYAML(t *testing.T) {
	idx := NewTreeSitterIndexer()
	content := []byte("name: demo\nmeta:\n  version: 1\n")
	hits, err := idx.IndexFile("a.yaml", content)
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, h := range hits {
		names[h.Name] = true
		if h.Kind != "key" {
			t.Fatalf("kind=%q", h.Kind)
		}
	}
	if !names["name"] || !names["meta"] || !names["version"] {
		t.Fatalf("hits=%+v", hits)
	}
}

func TestTreeSitterIndexerUnsupported(t *testing.T) {
	idx := NewTreeSitterIndexer()
	_, err := idx.IndexFile("a.md", []byte("# hi"))
	if err == nil {
		t.Fatal("expected unsupported error")
	}
}
