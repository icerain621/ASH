package artifacts

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ValidateQuality rejects known placeholder delivery artifacts when strict mode
// is enabled. Static smoke runs can call it with strict=false.
func ValidateQuality(runDir string, manifest *Manifest, strict bool) error {
	if manifest == nil {
		return fmt.Errorf("missing artifact manifest")
	}
	for _, art := range manifest.Artifacts {
		switch art.Type {
		case "diff":
			body, err := ReadManifestArtifact(runDir, art.URI)
			if err != nil {
				return fmt.Errorf("read diff artifact failed: %w", err)
			}
			if strict && IsPlaceholderDiff(body) {
				return fmt.Errorf("diff.patch is placeholder; real delivery did not produce a working tree diff")
			}
		case "test_report":
			body, err := ReadManifestArtifact(runDir, art.URI)
			if err != nil {
				return fmt.Errorf("read test_report artifact failed: %w", err)
			}
			if strict && IsPlaceholderTestReport(body) {
				return fmt.Errorf("test_report.json is placeholder; test.run did not produce a real report")
			}
		}
	}
	return nil
}

func ReadManifestArtifact(runDir, uri string) ([]byte, error) {
	if strings.HasPrefix(uri, "fs://") {
		return os.ReadFile(strings.TrimPrefix(uri, "fs://"))
	}
	rel := strings.TrimPrefix(filepath.ToSlash(uri), "artifacts/")
	if rel == filepath.ToSlash(uri) {
		rel = uri
	}
	if strings.Contains(rel, "..") || filepath.IsAbs(rel) {
		return nil, fmt.Errorf("unsafe artifact uri %q", uri)
	}
	return os.ReadFile(filepath.Join(runDir, "artifacts", filepath.FromSlash(rel)))
}

func IsPlaceholderDiff(body []byte) bool {
	text := strings.TrimSpace(string(body))
	if text == "" {
		return true
	}
	for _, marker := range []string{
		"No working tree diff was produced",
		"Static executor produced no code changes",
		"M0 stub patch",
	} {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}

func IsPlaceholderTestReport(body []byte) bool {
	var report map[string]any
	if err := json.Unmarshal(body, &report); err != nil {
		return false
	}
	errText, _ := report["error"].(string)
	return strings.Contains(errText, "test.run did not produce a report")
}
