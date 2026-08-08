package doctor

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

//go:embed testdata/probe-repo/*
var probeRepoFS embed.FS

// ProbeDatasetID identifies the fixed TR0 synthetic corpus (appendix E).
const ProbeDatasetID = "ash.doctor.probe/v1"

// materializeProbeRepo copies the embedded fixed corpus into destDir and writes
// a per-case CASE.md so issueOrSpec remains deterministic but case-scoped.
func materializeProbeRepo(destDir, caseID string) (issue string, err error) {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", err
	}
	root := "testdata/probe-repo"
	err = fs.WalkDir(probeRepoFS, root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		target := filepath.Join(destDir, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := probeRepoFS.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		return "", err
	}
	issue = "doctor " + strings.TrimSpace(caseID)
	caseBody := fmt.Sprintf("# Case %s\n\ndataset: %s\nissue: %s\n", caseID, ProbeDatasetID, issue)
	if err := os.WriteFile(filepath.Join(destDir, "CASE.md"), []byte(caseBody), 0o644); err != nil {
		return "", err
	}
	return issue, nil
}
