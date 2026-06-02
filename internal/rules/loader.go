package rules

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Loader loads and caches validated scenario documents from a directory.
type Loader struct {
	dir  string
	mu   sync.RWMutex
	byKey map[string]*Document // key: name@version
}

func NewLoader(dir string) *Loader {
	return &Loader{
		dir:   dir,
		byKey: make(map[string]*Document),
	}
}

func scenarioKey(name, version string) string {
	return name + "@" + version
}

// LoadDir reads all *.yaml / *.yml files from the scenarios directory.
func (l *Loader) LoadDir() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.byKey = make(map[string]*Document)

	if l.dir == "" {
		return fmt.Errorf("scenarios dir not configured")
	}
	info, err := os.Stat(l.dir)
	if err != nil {
		return fmt.Errorf("stat scenarios dir: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("scenarios path is not a directory: %s", l.dir)
	}

	entries, err := os.ReadDir(l.dir)
	if err != nil {
		return fmt.Errorf("read scenarios dir: %w", err)
	}

	var loadErrs []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			continue
		}
		path := filepath.Join(l.dir, name)
		raw, err := os.ReadFile(path)
		if err != nil {
			loadErrs = append(loadErrs, fmt.Sprintf("%s: read: %v", name, err))
			continue
		}
		res := ParseAndValidate(raw)
		if !res.OK {
			loadErrs = append(loadErrs, fmt.Sprintf("%s: invalid: %d issues", name, len(res.Issues)))
			continue
		}
		key := scenarioKey(res.Doc.Scenario.Name, res.Doc.Scenario.ScenarioVersion)
		l.byKey[key] = res.Doc
	}
	if len(loadErrs) > 0 {
		return fmt.Errorf("scenario load errors: %s", strings.Join(loadErrs, "; "))
	}
	return nil
}

// Get returns a scenario by name and version.
func (l *Loader) Get(name, version string) (*Document, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	doc, ok := l.byKey[scenarioKey(name, version)]
	if !ok {
		return nil, fmt.Errorf("scenario not found: %s@%s", name, version)
	}
	return doc, nil
}

// List returns summaries of loaded scenarios.
func (l *Loader) List() []ScenarioSummary {
	l.mu.RLock()
	defer l.mu.RUnlock()
	out := make([]ScenarioSummary, 0, len(l.byKey))
	for _, doc := range l.byKey {
		sc := doc.Scenario
		out = append(out, ScenarioSummary{
			Name:            sc.Name,
			ScenarioVersion: sc.ScenarioVersion,
			Description:     sc.Description,
			PolicyProfile:   sc.PolicyProfile,
			StepCount:       len(sc.Steps),
			GateCount:       len(sc.Gates),
		})
	}
	return out
}

// ValidateYAML validates arbitrary YAML without caching.
func (l *Loader) ValidateYAML(raw []byte) ValidationResult {
	return ParseAndValidate(raw)
}

// RawYAML loads file content for a scenario (for GET detail).
func (l *Loader) RawYAML(name, version string) ([]byte, error) {
	if l.dir == "" {
		return nil, fmt.Errorf("scenarios dir not configured")
	}
	entries, err := os.ReadDir(l.dir)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		path := filepath.Join(l.dir, e.Name())
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		res := ParseAndValidate(raw)
		if !res.OK || res.Doc == nil {
			continue
		}
		if res.Doc.Scenario.Name == name && res.Doc.Scenario.ScenarioVersion == version {
			return raw, nil
		}
	}
	return nil, fmt.Errorf("scenario file not found: %s@%s", name, version)
}
