package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScanAndContextRefs(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, ".ash", "skills", "ash-demo")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "---\nname: ash-demo\ndescription: Demo skill for tests. Use when testing.\n---\n\n# Demo\n\nDo the thing.\n"
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	list, err := ScanRepo(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Items) != 1 || list.Items[0].ID != "ash-demo" {
		t.Fatalf("%+v", list.Items)
	}
	if list.Items[0].ContextRef != "skill:ash-demo" {
		t.Fatalf("ref=%s", list.Items[0].ContextRef)
	}

	got, err := Get(dir, "ash-demo")
	if err != nil || !strings.Contains(got.Body, "Do the thing") {
		t.Fatalf("%v %+v", err, got)
	}

	refs := ContextRefsForWanted(dir, []string{"ash-demo", "missing"})
	if len(refs) != 1 || refs[0] != "skill:ash-demo" {
		t.Fatalf("%v", refs)
	}
	if ContextRefsForWanted(dir, nil) != nil {
		t.Fatal("expected nil")
	}
}
