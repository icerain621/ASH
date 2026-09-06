package rag

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ash-repwiki/ash/internal/store"
)

func withIsolatedLSPPool(t *testing.T) {
	t.Helper()
	prev := sharedLSPSessionPool
	sharedLSPSessionPool = newLSPSessionPool()
	t.Cleanup(func() {
		sharedLSPSessionPool.closeAll()
		sharedLSPSessionPool = prev
	})
}

func TestHoverDefinitionViaFakeGopls(t *testing.T) {
	bin := buildFakeGopls(t)
	t.Setenv(envRAGLSPGopls, bin)
	withIsolatedLSPPool(t)

	root := t.TempDir()
	code := "package main\n\n// pad\n\n\n\nfunc FixtureLSPFunc() {}\n"
	rel := "sample.go"
	if err := os.WriteFile(filepath.Join(root, rel), []byte(code), 0o644); err != nil {
		t.Fatal(err)
	}

	svc := NewService(nil)
	hover, err := svc.Hover(LSPPositionQuery{
		RepoRoot:  root,
		Path:      rel,
		Line:      7,
		Character: 6,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(hover.Contents, "FixtureLSPFunc") {
		t.Fatalf("contents=%q", hover.Contents)
	}
	if hover.Kind != "markdown" {
		t.Fatalf("kind=%q", hover.Kind)
	}
	if hover.Path != rel {
		t.Fatalf("path=%q", hover.Path)
	}

	def, err := svc.Definition(LSPPositionQuery{
		RepoRoot:  root,
		Path:      rel,
		Line:      7,
		Character: 6,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(def.Locations) != 1 {
		t.Fatalf("locations=%+v", def.Locations)
	}
	if def.Locations[0].Line != 7 {
		t.Fatalf("line=%d", def.Locations[0].Line)
	}
	if def.Locations[0].Path != rel {
		t.Fatalf("def path=%q want %q", def.Locations[0].Path, rel)
	}

	refs, err := svc.References(LSPReferencesRequest{
		LSPPositionQuery: LSPPositionQuery{
			RepoRoot:  root,
			Path:      rel,
			Line:      7,
			Character: 6,
		},
		Limit: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if refs.Source != "lsp" {
		t.Fatalf("source=%q", refs.Source)
	}
	if len(refs.Locations) != 2 {
		t.Fatalf("refs=%+v want 2", refs.Locations)
	}
}

func TestReferencesSymbolTableFallback(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	svc := NewService(db)
	root := t.TempDir()
	abs, err := AbsRepoRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	rel := "a.go"
	if err := os.WriteFile(filepath.Join(root, rel), []byte("package p\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	rows := []store.RAGSymbol{
		{ID: "s1", SpaceID: "fb", RepoRoot: abs, Path: rel, Name: "Alpha", Kind: "func", Line: 7, Digest: "d1", CreatedAt: now, UpdatedAt: now},
		{ID: "s2", SpaceID: "fb", RepoRoot: abs, Path: "b.go", Name: "Alpha", Kind: "func", Line: 3, Digest: "d2", CreatedAt: now, UpdatedAt: now},
	}
	for _, row := range rows {
		if err := db.Create(&row).Error; err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv(envRAGLSPGopls, "")
	t.Setenv(envRAGLSPTS, "")
	// Force unavailable LSP by pointing at missing binary after clearing PATH lookup — use unsupported via empty gopls.
	// prepareLSPQuery looks up gopls; with empty env and no PATH gopls, Indexer may still find system gopls.
	// Use a non-go file? Better: use .go with ASH_RAG_LSP_GOPLS pointing to missing path.
	t.Setenv(envRAGLSPGopls, filepath.Join(t.TempDir(), "missing-gopls"))

	refs, err := svc.References(LSPReferencesRequest{
		LSPPositionQuery: LSPPositionQuery{
			RepoRoot: root,
			Path:     rel,
			Line:     7,
			SpaceID:  "fb",
		},
		Limit: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if refs.Source != "symbol_table" {
		t.Fatalf("source=%q want symbol_table", refs.Source)
	}
	if len(refs.Locations) < 2 {
		t.Fatalf("locations=%+v", refs.Locations)
	}
}

func TestQueryExpandRefs(t *testing.T) {
	bin := buildFakeGopls(t)
	t.Setenv(envRAGLSPGopls, bin)
	withIsolatedLSPPool(t)

	db := store.OpenTest(t, t.TempDir())
	svc := NewService(db)
	root := t.TempDir()
	code := "package main\n\n// pad\n\n\n\nfunc FixtureLSPFunc() {}\n"
	rel := "sample.go"
	if err := os.WriteFile(filepath.Join(root, rel), []byte(code), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.RebuildSymbols(RebuildSymbolsRequest{RepoRoot: root, SpaceID: "ex"}); err != nil {
		t.Fatal(err)
	}
	resp, err := svc.Query(QueryRequest{
		RepoRoot:   root,
		Text:       "FixtureLSPFunc",
		SpaceID:    "ex",
		Prefer:     "symbol",
		ExpandRefs: true,
		TopK:       20,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Items) == 0 {
		t.Fatal("expected hits")
	}
	foundRef := false
	for _, h := range resp.Items {
		if strings.HasPrefix(h.Snippet, "ref ") {
			foundRef = true
			break
		}
	}
	if !foundRef {
		t.Fatalf("expected expanded ref hit, items=%+v", resp.Items)
	}
}

func TestParseHoverAndDefinition(t *testing.T) {
	h, err := parseHoverResult([]byte(`{"contents":{"kind":"markdown","value":"**X**"},"range":{"start":{"line":1,"character":0},"end":{"line":1,"character":1}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if h.Contents != "**X**" || h.Kind != "markdown" || h.Range == nil || h.Range.Start.Line != 1 {
		t.Fatalf("%+v", h)
	}
	locs, err := parseDefinitionResult([]byte(`{"uri":"file:///tmp/a.go","range":{"start":{"line":2,"character":3},"end":{"line":2,"character":4}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(locs) != 1 || locs[0].Line != 3 || locs[0].Character != 3 {
		t.Fatalf("%+v", locs)
	}
}

func TestLSPTimeoutAndMaxOpenEnv(t *testing.T) {
	t.Setenv(envRAGLSPTimeout, "5")
	if got := lspRequestTimeoutDuration(); got != 5*time.Second {
		t.Fatalf("timeout=%v", got)
	}
	t.Setenv(envRAGLSPTimeout, "0")
	if got := lspRequestTimeoutDuration(); got != defaultLSPTimeout {
		t.Fatalf("timeout fallback=%v", got)
	}
	t.Setenv(envRAGLSPMaxOpenDocs, "8")
	if got := lspMaxOpenDocs(); got != 8 {
		t.Fatalf("maxOpen=%d", got)
	}
}

func TestHoverMissingBinaryUnavailable(t *testing.T) {
	t.Setenv(envRAGLSPGopls, filepath.Join(t.TempDir(), "missing-gopls-bin"))
	t.Setenv(envRAGLSPTS, "")
	root := t.TempDir()
	rel := "a.go"
	if err := os.WriteFile(filepath.Join(root, rel), []byte("package p\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := NewService(nil).Hover(LSPPositionQuery{RepoRoot: root, Path: rel, Line: 1})
	if err == nil || !errors.Is(err, ErrLSPUnavailable) {
		t.Fatalf("err=%v want ErrLSPUnavailable", err)
	}
}
