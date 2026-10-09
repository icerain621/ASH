package toolbus

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCoreReadWriteEditStayInsideRoot(t *testing.T) {
	root := t.TempDir()
	bus := NewBus(DefaultRegistry())
	ctx := Context{RepoRoot: root}
	wrote := bus.Call(ctx, CallRequest{Tool: "write", Args: map[string]any{"path": "note.txt", "content": "alpha"}})
	if !wrote.OK {
		t.Fatalf("write=%+v", wrote)
	}
	edited := bus.Call(ctx, CallRequest{Tool: "edit", Args: map[string]any{"path": "note.txt", "old": "alpha", "new": "beta"}})
	if !edited.OK {
		t.Fatalf("edit=%+v", edited)
	}
	read := bus.Call(ctx, CallRequest{Tool: "read", Args: map[string]any{"path": "note.txt"}})
	if !read.OK || read.Output["content"] != "beta" {
		t.Fatalf("read=%+v", read)
	}
	escaped := bus.Call(ctx, CallRequest{Tool: "read", Args: map[string]any{"path": filepath.Join("..", "secret.txt")}})
	if escaped.OK {
		t.Fatal("expected path escape to fail")
	}
	body, err := os.ReadFile(filepath.Join(root, "note.txt"))
	if err != nil || string(body) != "beta" {
		t.Fatalf("body=%q err=%v", body, err)
	}
}

func TestRAGQueryUsesBoundQuerier(t *testing.T) {
	bus := NewBus(DefaultRegistry())
	calls := 0
	ctx := Context{
		RepoRoot: t.TempDir(),
		RAGQuery: func(text string, topK int) (map[string]any, error) {
			calls++
			return map[string]any{"items": []any{}, "count": 0, "q": text, "topK": topK}, nil
		},
	}
	first := bus.Call(ctx, CallRequest{Tool: "rag.query", Args: map[string]any{"text": "login"}})
	if !first.OK || calls != 1 {
		t.Fatalf("first=%+v calls=%d", first, calls)
	}
	second := bus.Call(ctx, CallRequest{Tool: "rag.query", Args: map[string]any{"query": "auth", "topK": 3}})
	if !second.OK || calls != 2 {
		t.Fatalf("second=%+v calls=%d", second, calls)
	}
	unbound := bus.Call(Context{RepoRoot: t.TempDir()}, CallRequest{Tool: "rag.query", Args: map[string]any{"text": "x"}})
	if unbound.OK {
		t.Fatal("expected unbound rag.query to fail")
	}
}

func TestBashRequiresApprovalBeforeRunner(t *testing.T) {
	called := false
	prev := runBash
	runBash = func(string, string) (string, error) {
		called = true
		return "ran", nil
	}
	t.Cleanup(func() { runBash = prev })

	bus := NewBus(DefaultRegistry())
	res := bus.Call(Context{RepoRoot: t.TempDir()}, CallRequest{Tool: "bash", Args: map[string]any{"command": "echo hi"}})
	if res.OK || called || !strings.Contains(res.Error, "approval") {
		t.Fatalf("res=%+v called=%v", res, called)
	}
}
