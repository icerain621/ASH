package artifacts

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
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
	StoreKey    string         `json:"storeKey,omitempty"`
	Digest      string         `json:"digest"`
	ContentType string         `json:"contentType"`
	SizeBytes   int64          `json:"sizeBytes"`
	Producer    map[string]any `json:"producer,omitempty"`
}

// Manifest is the required artifact index for a run.
type Manifest struct {
	RunID        string         `json:"runId"`
	Scenario     map[string]any `json:"scenario"`
	CreatedAt    int64          `json:"createdAt"`
	ContextRefs  []string       `json:"contextRefs,omitempty"`
	Artifacts    []Entry        `json:"artifacts"`
}

// BundleMeta identifies the run for manifest generation.
type BundleMeta struct {
	RunID           string
	ScenarioName    string
	ScenarioVersion string
	StepID          string
	Role            string
	RepoRoot        string
	Issue           string
	EventRange      string
	AgentTaskID     string
	EvidenceRefs    []string
}

// WriteBundle ensures four M0 artifacts exist and writes manifest.json.
func WriteBundle(runDir string, meta BundleMeta) (*Manifest, error) {
	if err := EnsureRunLayout(runDir); err != nil {
		return nil, err
	}
	artDir := filepath.Join(runDir, "artifacts")

	if err := ensureDiff(artDir, meta.RepoRoot); err != nil {
		return nil, err
	}
	if err := ensureTestReport(artDir); err != nil {
		return nil, err
	}
	release := buildReleaseNotes(meta, artDir)
	rollback := buildRollbackPlan(meta, runDir)

	files := []struct {
		id, typ, name, rel, ctype, content string
	}{
		{"art_diff", "diff", "diff.patch", "diff.patch", "text/plain", ""},
		{"art_test", "test_report", "test_report.json", "test_report.json", "application/json", ""},
		{"art_release", "release_notes", "release_notes.md", "release_notes.md", "text/markdown", release},
		{"art_rollback", "rollback_plan", "rollback_plan.md", "rollback_plan.md", "text/markdown", rollback},
	}

	_ = os.WriteFile(filepath.Join(artDir, "release_notes.md"), []byte(normalizeLF(release)), DefaultFilePerm)
	_ = os.WriteFile(filepath.Join(artDir, "rollback_plan.md"), []byte(normalizeLF(rollback)), DefaultFilePerm)

	manifest := &Manifest{
		RunID: meta.RunID,
		Scenario: map[string]any{
			"name":            meta.ScenarioName,
			"scenarioVersion": meta.ScenarioVersion,
		},
		CreatedAt:   time.Now().UTC().UnixMilli(),
		ContextRefs: append([]string(nil), meta.EvidenceRefs...),
	}

	coreProducer := map[string]any{
		"stepId":       meta.StepID,
		"role":         meta.Role,
		"eventRange":   meta.EventRange,
		"agentTaskId":  meta.AgentTaskID,
		"evidenceRefs": meta.EvidenceRefs,
	}
	for _, f := range files {
		if err := appendManifestEntry(manifest, artDir, f.id, f.typ, f.name, f.rel, f.ctype, coreProducer); err != nil {
			return nil, err
		}
	}

	stepFiles, _ := filepath.Glob(filepath.Join(artDir, "*.md"))
	for _, abs := range stepFiles {
		name := filepath.Base(abs)
		if name == "release_notes.md" || name == "rollback_plan.md" {
			continue
		}
		stepID := inferStepIDFromArtifactName(name)
		producer := map[string]any{
			"stepId":       stepID,
			"role":         roleFromStepArtifact(abs),
			"eventRange":   meta.EventRange,
			"evidenceRefs": meta.EvidenceRefs,
		}
		id := "art_step_" + strings.TrimSuffix(name, filepath.Ext(name))
		if err := appendManifestEntry(manifest, artDir, id, "step_output", name, name, "text/markdown", producer); err != nil {
			return nil, err
		}
	}

	if err := SaveManifest(runDir, manifest); err != nil {
		return nil, err
	}
	if err := enforceArtifactsBudget(artDir); err != nil {
		return nil, err
	}
	return manifest, nil
}

func appendManifestEntry(manifest *Manifest, artDir, id, typ, name, rel, ctype string, producer map[string]any) error {
	abs := filepath.Join(artDir, rel)
	b, err := os.ReadFile(abs)
	if err != nil {
		return err
	}
	digestSrc := b
	if ctype == "application/json" {
		var node any
		if err := json.Unmarshal(b, &node); err == nil {
			if cb, err := MarshalCanonicalJSON(node); err == nil {
				digestSrc = cb
			}
		}
	}
	manifest.Artifacts = append(manifest.Artifacts, Entry{
		ID:          id,
		Type:        typ,
		Name:        name,
		URI:         "artifacts/" + filepath.ToSlash(rel),
		Digest:      DigestBytes(digestSrc),
		ContentType: ctype,
		SizeBytes:   int64(len(b)),
		Producer:    producer,
	})
	return nil
}

func SaveManifest(runDir string, manifest *Manifest) error {
	out, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	text := normalizeLF(string(out)) + "\n"
	return os.WriteFile(filepath.Join(runDir, "artifacts", "manifest.json"), []byte(text), DefaultFilePerm)
}

func ensureDiff(artDir, repoRoot string) error {
	path := filepath.Join(artDir, "diff.patch")
	if st, err := os.Stat(path); err == nil && st.Size() > 0 {
		return nil
	}
	diff := ""
	if repoRoot != "" && isGitWorkTree(repoRoot) {
		out, err := exec.Command("git", "-C", repoRoot, "diff", "HEAD", "--binary").CombinedOutput()
		if err != nil {
			return fmt.Errorf("git diff: %w (%s)", err, strings.TrimSpace(string(out)))
		}
		diff = string(out)
	}
	if strings.TrimSpace(diff) == "" {
		diff = "# No working tree diff was produced.\n"
	}
	return os.WriteFile(path, []byte(normalizeLF(diff)), DefaultFilePerm)
}

func ensureTestReport(artDir string) error {
	path := filepath.Join(artDir, "test_report.json")
	if st, err := os.Stat(path); err == nil && st.Size() > 0 {
		return nil
	}
	report := map[string]any{
		"ok":      false,
		"summary": map[string]any{"passed": 0, "failed": 0, "skipped": 0},
		"error":   "test.run did not produce a report",
	}
	b, err := MarshalCanonicalJSON(report)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, DefaultFilePerm)
}

func enforceArtifactsBudget(artDir string) error {
	max := MaxArtifactsBytes()
	if max <= 0 {
		return nil
	}
	var total int64
	entries, err := os.ReadDir(artDir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		total += info.Size()
	}
	if total > max {
		return fmt.Errorf("artifacts total size %d exceeds ASH_ARTIFACTS_MAX_BYTES=%d", total, max)
	}
	return nil
}

func inferStepIDFromArtifactName(name string) string {
	base := strings.TrimSuffix(name, filepath.Ext(name))
	return strings.ReplaceAll(base, "_", ".")
}

func roleFromStepArtifact(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "# ") || !strings.HasSuffix(line, " step") {
			continue
		}
		return strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "# "), " step"))
	}
	return ""
}

func buildReleaseNotes(meta BundleMeta, artDir string) string {
	testStatus := "unknown"
	if b, err := os.ReadFile(filepath.Join(artDir, "test_report.json")); err == nil {
		var obj map[string]any
		if json.Unmarshal(b, &obj) == nil {
			testStatus = fmt.Sprintf("%v", obj["ok"])
		}
	}
	var b strings.Builder
	b.WriteString("# Release notes\n\n")
	b.WriteString(fmt.Sprintf("- Run: `%s`\n", meta.RunID))
	b.WriteString(fmt.Sprintf("- Scenario: `%s@%s`\n", meta.ScenarioName, meta.ScenarioVersion))
	if meta.Issue != "" {
		b.WriteString(fmt.Sprintf("- Issue/spec: %s\n", oneLine(meta.Issue)))
	}
	if meta.AgentTaskID != "" {
		b.WriteString(fmt.Sprintf("- Agent task: `%s`\n", meta.AgentTaskID))
	}
	b.WriteString(fmt.Sprintf("- Test status: `%s`\n", testStatus))
	if len(meta.EvidenceRefs) > 0 {
		b.WriteString("\n## Evidence\n\n")
		for _, ref := range meta.EvidenceRefs {
			b.WriteString("- `" + ref + "`\n")
		}
	}
	b.WriteString("\n## Notes\n\nReview `diff.patch` and `test_report.json` before merge or release.\n")
	return b.String()
}

func buildRollbackPlan(meta BundleMeta, runDir string) string {
	var b strings.Builder
	b.WriteString("# Rollback plan\n\n")
	b.WriteString("1. Stop rollout for this run if it has started.\n")
	b.WriteString("2. Revert or discard the changes represented by `artifacts/diff.patch`.\n")
	if meta.RepoRoot != "" {
		b.WriteString(fmt.Sprintf("3. Restore repository `%s` to the pre-run revision or previous deployment tag.\n", meta.RepoRoot))
	} else {
		b.WriteString("3. Restore the target repository to the pre-run revision or previous deployment tag.\n")
	}
	b.WriteString(fmt.Sprintf("4. Use checkpoints and event evidence under `%s` to audit what changed.\n", runDir))
	b.WriteString("5. Re-run the scenario in `replay` mode after rollback to verify recovery.\n")
	return b.String()
}

func isGitWorkTree(root string) bool {
	out, err := exec.Command("git", "-C", root, "rev-parse", "--is-inside-work-tree").CombinedOutput()
	return err == nil && strings.TrimSpace(string(out)) == "true"
}

func oneLine(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) <= 220 {
		return s
	}
	return s[:220] + "..."
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

func normalizeLF(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "\n"), "\r", "\n")
}
