package sandbox_test

import (
	"path/filepath"
	"testing"

	"github.com/ash-repwiki/ash/internal/sandbox"
)

func TestPathWithinRoot(t *testing.T) {
	root := t.TempDir()
	ok, err := sandbox.PathWithinRoot(root, filepath.Join(root, "a.txt"))
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	ok, err = sandbox.PathWithinRoot(root, filepath.Join(root, "..", "outside.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected path outside root rejected")
	}
}
