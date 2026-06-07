package openapicheck

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

var httpMethods = map[string]bool{
	"get": true, "post": true, "put": true, "patch": true, "delete": true,
	"head": true, "options": true, "trace": true,
}

// PathMethods maps OpenAPI path -> HTTP method (lowercase).
type PathMethods map[string]map[string]struct{}

// LoadPathMethods reads path/method pairs from an OpenAPI 3 YAML file.
func LoadPathMethods(path string) (PathMethods, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var root map[string]any
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, err
	}
	rawPaths, _ := root["paths"].(map[string]any)
	out := PathMethods{}
	for p, raw := range rawPaths {
		ops, _ := raw.(map[string]any)
		out[p] = map[string]struct{}{}
		for k := range ops {
			kl := strings.ToLower(k)
			if httpMethods[kl] {
				out[p][kl] = struct{}{}
			}
		}
	}
	return out, nil
}

// AlignReport summarizes contract vs implementation drift.
type AlignReport struct {
	Missing       []string
	LegacyPlanned []string
	Undocumented  []string
}

// AlignContract checks that enforced contract paths exist in swagger output.
// Only paths under enforcedPrefix are required; legacyPlannedPrefix paths are
// reported separately and do not fail the check.
func AlignContract(contract, swagger PathMethods, enforcedPrefix, legacyPlannedPrefix string) AlignReport {
	var rep AlignReport
	for path, methods := range contract {
		if legacyPlannedPrefix != "" && strings.HasPrefix(path, legacyPlannedPrefix) {
			for m := range methods {
				rep.LegacyPlanned = append(rep.LegacyPlanned, fmt.Sprintf("%s %s", path, strings.ToUpper(m)))
			}
			continue
		}
		if !strings.HasPrefix(path, enforcedPrefix) {
			continue
		}
		swMethods, ok := swagger[path]
		if !ok {
			for m := range methods {
				rep.Missing = append(rep.Missing, fmt.Sprintf("%s %s (path missing)", path, strings.ToUpper(m)))
			}
			continue
		}
		for m := range methods {
			if _, ok := swMethods[m]; !ok {
				rep.Missing = append(rep.Missing, fmt.Sprintf("%s %s", path, strings.ToUpper(m)))
			}
		}
	}
	for path, methods := range swagger {
		if !strings.HasPrefix(path, enforcedPrefix) {
			continue
		}
		if _, inContract := contract[path]; inContract {
			continue
		}
		for m := range methods {
			rep.Undocumented = append(rep.Undocumented, fmt.Sprintf("%s %s", path, strings.ToUpper(m)))
		}
	}
	sort.Strings(rep.Missing)
	sort.Strings(rep.LegacyPlanned)
	sort.Strings(rep.Undocumented)
	return rep
}

// FormatReport returns a human-readable summary.
func FormatReport(rep AlignReport) string {
	var b strings.Builder
	if n := len(rep.Missing); n > 0 {
		fmt.Fprintf(&b, "missing from swagger (%d):\n%s\n", n, strings.Join(rep.Missing, "\n"))
	}
	if n := len(rep.LegacyPlanned); n > 0 {
		fmt.Fprintf(&b, "legacy planned paths (%d, informational):\n%s\n", n, strings.Join(rep.LegacyPlanned, "\n"))
	}
	if n := len(rep.Undocumented); n > 0 {
		fmt.Fprintf(&b, "implemented but not in contract draft (%d, informational):\n%s\n", n, strings.Join(rep.Undocumented, "\n"))
	}
	return strings.TrimSpace(b.String())
}
