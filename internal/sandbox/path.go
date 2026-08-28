package sandbox

import (
	"fmt"
	"path/filepath"
	"strings"
)

// PathWithinRoot reports whether candidate resolves under root (M4-SBX-03).
func PathWithinRoot(root, candidate string) (bool, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return false, fmt.Errorf("repoRoot is required")
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return false, err
	}
	absRoot = filepath.Clean(absRoot)
	absCand, err := filepath.Abs(candidate)
	if err != nil {
		return false, err
	}
	absCand = filepath.Clean(absCand)
	rel, err := filepath.Rel(absRoot, absCand)
	if err != nil {
		return false, err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false, nil
	}
	return true, nil
}
