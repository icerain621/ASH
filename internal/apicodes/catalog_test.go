package apicodes

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func TestHandlerErrorCodesAreCatalogued(t *testing.T) {
	root := repoRoot(t)
	used, err := ScanHandlerCodes(filepath.Join(root, "internal", "api"))
	if err != nil {
		t.Fatal(err)
	}
	var missing, orphan []string
	for _, code := range used {
		if _, ok := Catalog[code]; !ok {
			missing = append(missing, code)
		}
	}
	usedSet := map[string]struct{}{}
	for _, code := range used {
		usedSet[code] = struct{}{}
	}
	for code := range Catalog {
		if _, ok := usedSet[code]; !ok {
			orphan = append(orphan, code)
		}
	}
	if len(missing) > 0 {
		t.Fatalf("handler codes missing from apicodes.Catalog:\n%s", strings.Join(missing, "\n"))
	}
	if len(orphan) > 0 {
		t.Fatalf("unused catalog codes (remove or wire handlers):\n%s", strings.Join(orphan, "\n"))
	}
}
