package knowledge

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// RepoProfile is an ephemeral scan of a repository root.
type RepoProfile struct {
	ID           string            `json:"id"`
	RepoRoot     string            `json:"repoRoot"`
	Languages    []string          `json:"languages"`
	Modules      []string          `json:"modules"`
	TestCommands []string          `json:"testCommands"`
	Markers      map[string]bool   `json:"markers"`
	Summary      string            `json:"summary"`
	ContextRef   string            `json:"contextRef"`
}

// BuildProfile walks repoRoot (shallow) and returns a heuristic profile.
func BuildProfile(repoRoot string) (*RepoProfile, error) {
	root := strings.TrimSpace(repoRoot)
	if root == "" {
		root = "."
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		abs = root
	}
	entries, err := os.ReadDir(abs)
	if err != nil {
		return nil, err
	}

	markers := map[string]bool{}
	langSet := map[string]struct{}{}
	var modules []string
	var tests []string

	for _, e := range entries {
		name := e.Name()
		lower := strings.ToLower(name)
		switch {
		case lower == "go.mod":
			markers["go"] = true
			langSet["go"] = struct{}{}
			tests = appendUnique(tests, "go test ./...")
			if mod := readGoModule(filepath.Join(abs, name)); mod != "" {
				modules = append(modules, mod)
			}
		case lower == "package.json":
			markers["node"] = true
			langSet["typescript"] = struct{}{}
			langSet["javascript"] = struct{}{}
			tests = appendUnique(tests, "npm test")
			modules = append(modules, "package.json")
		case lower == "pyproject.toml" || lower == "requirements.txt" || lower == "setup.py":
			markers["python"] = true
			langSet["python"] = struct{}{}
			tests = appendUnique(tests, "pytest")
		case lower == "cargo.toml":
			markers["rust"] = true
			langSet["rust"] = struct{}{}
			tests = appendUnique(tests, "cargo test")
		case lower == "pom.xml" || lower == "build.gradle" || lower == "build.gradle.kts":
			markers["jvm"] = true
			langSet["java"] = struct{}{}
			tests = appendUnique(tests, "mvn test")
		case lower == "makefile":
			markers["make"] = true
			tests = appendUnique(tests, "make test")
		case strings.HasPrefix(lower, "dockerfile"):
			markers["docker"] = true
		case e.IsDir() && (lower == "cmd" || lower == "internal" || lower == "pkg" || lower == "src" || lower == "frontend" || lower == "backend"):
			modules = append(modules, name)
		}
	}

	// Extension sampling (top-level files only for B ephemeral speed)
	_ = filepath.WalkDir(abs, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			if d != nil && d.IsDir() {
				base := d.Name()
				if base == ".git" || base == "node_modules" || base == "vendor" || base == "dist" || base == ".ash" {
					return filepath.SkipDir
				}
				// limit depth
				rel, _ := filepath.Rel(abs, path)
				if rel != "." && strings.Count(rel, string(os.PathSeparator)) >= 2 {
					return filepath.SkipDir
				}
			}
			return nil
		}
		switch strings.ToLower(filepath.Ext(d.Name())) {
		case ".go":
			langSet["go"] = struct{}{}
		case ".ts", ".tsx":
			langSet["typescript"] = struct{}{}
		case ".js", ".jsx":
			langSet["javascript"] = struct{}{}
		case ".py":
			langSet["python"] = struct{}{}
		case ".rs":
			langSet["rust"] = struct{}{}
		case ".java", ".kt":
			langSet["java"] = struct{}{}
		}
		return nil
	})

	langs := keys(langSet)
	sort.Strings(langs)
	sort.Strings(modules)
	sort.Strings(tests)
	sum := sha256.Sum256([]byte(abs + "|" + strings.Join(langs, ",") + "|" + strings.Join(modules, ",")))
	id := "profile_" + hex.EncodeToString(sum[:8])
	summary := "languages=" + strings.Join(langs, ",")
	if len(modules) > 0 {
		summary += "; modules=" + strings.Join(modules, ",")
	}
	return &RepoProfile{
		ID: id, RepoRoot: abs, Languages: langs, Modules: modules, TestCommands: tests,
		Markers: markers, Summary: summary, ContextRef: "profile:" + id,
	}, nil
}

func readGoModule(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}
	return ""
}

func keys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func appendUnique(list []string, v string) []string {
	for _, x := range list {
		if x == v {
			return list
		}
	}
	return append(list, v)
}
