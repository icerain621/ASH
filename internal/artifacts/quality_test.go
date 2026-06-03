package artifacts

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateQualityRejectsPlaceholdersInStrictMode(t *testing.T) {
	runDir := t.TempDir()
	artDir := filepath.Join(runDir, "artifacts")
	if err := os.MkdirAll(artDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(artDir, "diff.patch"), []byte("# M0 stub patch\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(artDir, "test_report.json"), []byte(`{"ok":false,"error":"test.run did not produce a report"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := &Manifest{Artifacts: []Entry{
		{Type: "diff", Name: "diff.patch", URI: "artifacts/diff.patch"},
		{Type: "test_report", Name: "test_report.json", URI: "artifacts/test_report.json"},
	}}

	if err := ValidateQuality(runDir, manifest, true); err == nil {
		t.Fatal("expected placeholder diff rejection")
	}

	if err := os.WriteFile(filepath.Join(artDir, "diff.patch"), []byte("diff --git a/README.md b/README.md\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := ValidateQuality(runDir, manifest, true); err == nil {
		t.Fatal("expected placeholder test_report rejection")
	}
}

func TestValidateQualitySupportsFSURI(t *testing.T) {
	runDir := t.TempDir()
	objectPath := filepath.Join(t.TempDir(), "diff.patch")
	if err := os.WriteFile(objectPath, []byte("diff --git a/README.md b/README.md\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	artDir := filepath.Join(runDir, "artifacts")
	if err := os.MkdirAll(artDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(artDir, "test_report.json"), []byte(`{"ok":true}`), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := &Manifest{Artifacts: []Entry{
		{Type: "diff", Name: "diff.patch", URI: "fs://" + objectPath},
		{Type: "test_report", Name: "test_report.json", URI: "artifacts/test_report.json"},
	}}

	if err := ValidateQuality(runDir, manifest, true); err != nil {
		t.Fatalf("ValidateQuality() error=%v", err)
	}
}
