package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Load resolves the observability config file (if any) and returns a validated document.
// When no file is found, returns Default() without error.
func Load() (Document, error) {
	path, err := ResolvePath()
	if err != nil {
		return Document{}, err
	}
	if path == "" {
		return Default(), nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return Document{}, fmt.Errorf("read observability config %s: %w", path, err)
	}
	res := ParseAndValidate(raw)
	if !res.OK {
		return Document{}, fmt.Errorf("invalid observability config %s: %s", path, firstIssue(res))
	}
	return *res.Doc, nil
}

// ResolvePath returns the config file path from ASH_OBSERVABILITY_CONFIG or well-known locations.
func ResolvePath() (string, error) {
	if p := strings.TrimSpace(os.Getenv("ASH_OBSERVABILITY_CONFIG")); p != "" {
		return p, nil
	}
	candidates := []string{"config/ash-observability.yaml"}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		candidates = append(candidates, filepath.Join(home, ".ash", "observability.yaml"))
	}
	for _, p := range candidates {
		if fileExists(p) {
			return p, nil
		}
	}
	return "", nil
}

func firstIssue(res ValidationResult) string {
	if len(res.Issues) == 0 {
		return "unknown validation error"
	}
	i := res.Issues[0]
	if i.Path != "" {
		return fmt.Sprintf("%s: %s", i.Path, i.Message)
	}
	return i.Message
}

func fileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}
