package artifacts

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Entry describes one artifact in manifest.
type Entry struct {
	ID          string         `json:"id"`
	Type        string         `json:"type"`
	Name        string         `json:"name,omitempty"`
	URI         string         `json:"uri"`
	Digest      string         `json:"digest"`
	ContentType string         `json:"contentType"`
	SizeBytes   int64          `json:"sizeBytes"`
	Producer    map[string]any `json:"producer,omitempty"`
}

// Manifest is the required artifact index for a run.
type Manifest struct {
	RunID     string         `json:"runId"`
	Scenario  map[string]any `json:"scenario"`
	CreatedAt int64          `json:"createdAt"`
	Artifacts []Entry        `json:"artifacts"`
}

// BundleMeta identifies the run for manifest generation.
type BundleMeta struct {
	RunID           string
	ScenarioName    string
	ScenarioVersion string
	StepID          string
	Role            string
}

// WriteBundle ensures four M0 artifacts exist and writes manifest.json.
func WriteBundle(runDir string, meta BundleMeta) (*Manifest, error) {
	artDir := filepath.Join(runDir, "artifacts")
	if err := os.MkdirAll(artDir, 0o755); err != nil {
		return nil, err
	}

	issue := meta.RunID
	release := fmt.Sprintf("# Release notes (M0 stub)\n\nRun `%s` completed for scenario %s@%s.\n", meta.RunID, meta.ScenarioName, meta.ScenarioVersion)
	rollback := fmt.Sprintf("# Rollback plan (M0 stub)\n\n1. Revert branch changes\n2. Restore from checkpoint in `%s/checkpoints/`\n", runDir)

	files := []struct {
		id, typ, name, rel, ctype, content string
	}{
		{"art_diff", "diff", "diff.patch", "diff.patch", "text/plain", "# M0 stub diff\n"},
		{"art_test", "test_report", "test_report.json", "test_report.json", "application/json", `{"ok":true,"summary":{"passed":0,"failed":0,"skipped":0}}`},
		{"art_release", "release_notes", "release_notes.md", "release_notes.md", "text/markdown", release},
		{"art_rollback", "rollback_plan", "rollback_plan.md", "rollback_plan.md", "text/markdown", rollback},
	}

	// Preserve diff/test_report if tools already wrote them.
	diffPath := filepath.Join(artDir, "diff.patch")
	if st, err := os.Stat(diffPath); err != nil || st.Size() == 0 {
		_ = os.WriteFile(diffPath, []byte(normalizeLF(files[0].content)), 0o644)
	}
	testPath := filepath.Join(artDir, "test_report.json")
	if st, err := os.Stat(testPath); err != nil || st.Size() == 0 {
		_ = os.WriteFile(testPath, []byte(files[1].content), 0o644)
	}
	_ = os.WriteFile(filepath.Join(artDir, "release_notes.md"), []byte(normalizeLF(release)), 0o644)
	_ = os.WriteFile(filepath.Join(artDir, "rollback_plan.md"), []byte(normalizeLF(rollback)), 0o644)
	_ = issue

	manifest := &Manifest{
		RunID: meta.RunID,
		Scenario: map[string]any{
			"name":            meta.ScenarioName,
			"scenarioVersion": meta.ScenarioVersion,
		},
		CreatedAt: time.Now().UTC().UnixMilli(),
	}

	for _, f := range files {
		abs := filepath.Join(artDir, f.rel)
		b, err := os.ReadFile(abs)
		if err != nil {
			return nil, err
		}
		manifest.Artifacts = append(manifest.Artifacts, Entry{
			ID:          f.id,
			Type:        f.typ,
			Name:        f.name,
			URI:         "artifacts/" + f.rel,
			Digest:      digestBytes(b),
			ContentType: f.ctype,
			SizeBytes:   int64(len(b)),
			Producer: map[string]any{
				"stepId": meta.StepID,
				"role":   meta.Role,
			},
		})
	}

	out, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, err
	}
	text := normalizeLF(string(out)) + "\n"
	if err := os.WriteFile(filepath.Join(artDir, "manifest.json"), []byte(text), 0o644); err != nil {
		return nil, err
	}
	return manifest, nil
}

// LoadManifest reads manifest.json from a run directory.
func LoadManifest(runDir string) (*Manifest, error) {
	b, err := os.ReadFile(filepath.Join(runDir, "artifacts", "manifest.json"))
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

func digestBytes(b []byte) string {
	h := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(h[:])
}

func normalizeLF(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "\n"), "\r", "\n")
}
