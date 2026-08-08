package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMaterializeProbeRepoFixedCorpus(t *testing.T) {
	dir := t.TempDir()
	issue, err := materializeProbeRepo(dir, "TR0-01")
	if err != nil {
		t.Fatal(err)
	}
	if issue != "doctor TR0-01" {
		t.Fatalf("issue=%q", issue)
	}
	for _, rel := range []string{"PROBE_MANIFEST.md", "README.md", "docs/SPEC.md", "src/service.go.txt", "CASE.md"} {
		path := filepath.Join(dir, filepath.FromSlash(rel))
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("missing %s: %v", rel, err)
		}
		if len(strings.TrimSpace(string(data))) == 0 {
			t.Fatalf("empty %s", rel)
		}
	}
	manifest, err := os.ReadFile(filepath.Join(dir, "PROBE_MANIFEST.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(manifest), ProbeDatasetID) {
		t.Fatalf("manifest missing dataset id %s", ProbeDatasetID)
	}
	caseBody, err := os.ReadFile(filepath.Join(dir, "CASE.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(caseBody), ProbeDatasetID) || !strings.Contains(string(caseBody), "TR0-01") {
		t.Fatalf("CASE.md=%s", caseBody)
	}
}
