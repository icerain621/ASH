package knowledge

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestContextRefsForRun_includesProfile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/x\n\ngo 1.22\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := NewService(nil, nil)
	refs := s.ContextRefsForRun("local", dir, "testing")
	if len(refs) == 0 {
		t.Fatal("expected refs")
	}
	foundProfile, foundWiki := false, false
	for _, r := range refs {
		if strings.HasPrefix(r, "profile:") {
			foundProfile = true
		}
		if strings.HasPrefix(r, "wiki:") {
			foundWiki = true
		}
	}
	if !foundProfile {
		t.Fatalf("missing profile ref in %v", refs)
	}
	if !foundWiki {
		t.Fatalf("missing wiki ref in %v", refs)
	}
}
