package apicodes

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
)

var errorBodyRE = regexp.MustCompile(`errorBody\("([A-Z][A-Z0-9_]*)"`)

// ScanHandlerCodes returns error codes referenced from internal/api handlers.
func ScanHandlerCodes(apiDir string) ([]string, error) {
	seen := map[string]struct{}{}
	err := filepath.Walk(apiDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || filepath.Ext(path) != ".go" || filepath.Base(path) == "docs.go" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, m := range errorBodyRE.FindAllStringSubmatch(string(data), -1) {
			seen[m[1]] = struct{}{}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(seen))
	for code := range seen {
		out = append(out, code)
	}
	sort.Strings(out)
	return out, nil
}
