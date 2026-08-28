package knowledge

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildProfile_goModule(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/app\n\ngo 1.22\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_ = os.Mkdir(filepath.Join(dir, "internal"), 0o755)
	_ = os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0o644)

	p, err := BuildProfile(dir)
	if err != nil {
		t.Fatal(err)
	}
	if p.ContextRef == "" || p.ID == "" {
		t.Fatal("expected id/contextRef")
	}
	foundGo := false
	for _, l := range p.Languages {
		if l == "go" {
			foundGo = true
		}
	}
	if !foundGo {
		t.Fatalf("languages=%v", p.Languages)
	}
	if len(p.TestCommands) == 0 {
		t.Fatal("expected test commands")
	}
}
