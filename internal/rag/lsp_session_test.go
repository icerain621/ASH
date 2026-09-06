package rag

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func countEventLogLines(t *testing.T, path, event string) int {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0
		}
		t.Fatal(err)
	}
	n := 0
	for _, line := range strings.Split(string(b), "\n") {
		if strings.TrimSpace(line) == event {
			n++
		}
	}
	return n
}

func TestLSPSessionReusesInitialize(t *testing.T) {
	bin := buildFakeGopls(t)
	logPath := filepath.Join(t.TempDir(), "events.log")
	t.Setenv("ASH_FAKE_GOPLS_EVENT_LOG", logPath)
	t.Setenv(envRAGLSPSession, "1")
	t.Setenv(envRAGLSPIdleSec, "60")

	pool := newLSPSessionPool()
	defer pool.closeAll()
	idx := NewLSPIndexer(bin, "")
	idx.pool = pool
	root := t.TempDir()
	idx.SetWorkspaceRoot(root)

	content1 := []byte("package main\n\n// pad\n\n\n\nfunc FixtureLSPFunc() {}\n")
	content2 := []byte("package main\n\n// pad\n\n\n\nfunc FixtureLSPFunc() {}\n")
	f1 := filepath.Join(root, "a.go")
	f2 := filepath.Join(root, "b.go")
	if err := os.WriteFile(f1, content1, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f2, content2, 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := idx.IndexFile(f1, content1); err != nil {
		t.Fatal(err)
	}
	if _, err := idx.IndexFile(f2, content2); err != nil {
		t.Fatal(err)
	}
	if got := countEventLogLines(t, logPath, "initialize"); got != 1 {
		t.Fatalf("initialize count=%d want 1 (session reuse)", got)
	}
	if got := countEventLogLines(t, logPath, "documentSymbol"); got != 2 {
		t.Fatalf("documentSymbol count=%d want 2", got)
	}
}

func TestLSPSessionIdleReclaim(t *testing.T) {
	bin := buildFakeGopls(t)
	logPath := filepath.Join(t.TempDir(), "events.log")
	t.Setenv("ASH_FAKE_GOPLS_EVENT_LOG", logPath)
	t.Setenv(envRAGLSPSession, "1")
	t.Setenv(envRAGLSPIdleSec, "1")

	pool := newLSPSessionPool()
	defer pool.closeAll()
	idx := NewLSPIndexer(bin, "")
	idx.pool = pool
	root := t.TempDir()
	idx.SetWorkspaceRoot(root)

	content := []byte("package main\n\n// pad\n\n\n\nfunc FixtureLSPFunc() {}\n")
	f := filepath.Join(root, "a.go")
	if err := os.WriteFile(f, content, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := idx.IndexFile(f, content); err != nil {
		t.Fatal(err)
	}
	time.Sleep(1100 * time.Millisecond)
	if _, err := idx.IndexFile(f, content); err != nil {
		t.Fatal(err)
	}
	if got := countEventLogLines(t, logPath, "initialize"); got != 2 {
		t.Fatalf("initialize count=%d want 2 after idle reclaim", got)
	}
}

func TestLSPSessionDisabledOneShot(t *testing.T) {
	bin := buildFakeGopls(t)
	logPath := filepath.Join(t.TempDir(), "events.log")
	t.Setenv("ASH_FAKE_GOPLS_EVENT_LOG", logPath)
	t.Setenv(envRAGLSPSession, "0")

	idx := NewLSPIndexer(bin, "")
	idx.pool = newLSPSessionPool()
	root := t.TempDir()
	idx.SetWorkspaceRoot(root)

	content := []byte("package main\n\n// pad\n\n\n\nfunc FixtureLSPFunc() {}\n")
	f1 := filepath.Join(root, "a.go")
	f2 := filepath.Join(root, "b.go")
	if err := os.WriteFile(f1, content, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f2, content, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := idx.IndexFile(f1, content); err != nil {
		t.Fatal(err)
	}
	if _, err := idx.IndexFile(f2, content); err != nil {
		t.Fatal(err)
	}
	if got := countEventLogLines(t, logPath, "initialize"); got != 2 {
		t.Fatalf("initialize count=%d want 2 in one-shot mode", got)
	}
}
