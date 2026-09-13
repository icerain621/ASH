package scoring

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ScenarioEvalProfile weights the four evaluation dimensions for a scenario (GV06–07).
type ScenarioEvalProfile struct {
	ID        string             `json:"id" yaml:"id"`
	Scenario  string             `json:"scenario" yaml:"scenario"`
	Label     string             `json:"label" yaml:"label"`
	Weights   map[string]float64 `json:"weights" yaml:"weights"`
	GateHints []string           `json:"gateHints,omitempty" yaml:"gateHints,omitempty"`
}

type profilesFile struct {
	Profiles []ScenarioEvalProfile `yaml:"profiles"`
}

// DefaultProfiles returns built-in ScenarioEvalProfile for the three stock scenarios.
func DefaultProfiles() []ScenarioEvalProfile {
	return []ScenarioEvalProfile{
		{
			ID: "feature_delivery", Scenario: "feature_delivery", Label: "特性交付",
			Weights: map[string]float64{
				"quality": 0.35, "safety": 0.15, "efficiency": 0.30, "governance": 0.20,
			},
			GateHints: []string{"artifact completeness", "citation coverage"},
		},
		{
			ID: "hotfix", Scenario: "hotfix", Label: "热修",
			Weights: map[string]float64{
				"quality": 0.25, "safety": 0.35, "efficiency": 0.25, "governance": 0.15,
			},
			GateHints: []string{"human approval wait", "fast verify"},
		},
		{
			ID: "security_patch", Scenario: "security_patch", Label: "安全补丁",
			Weights: map[string]float64{
				"quality": 0.20, "safety": 0.45, "efficiency": 0.10, "governance": 0.25,
			},
			GateHints: []string{"notice completeness", "rollback readiness"},
		},
	}
}

// LoadProfiles loads YAML from path, or returns defaults when missing/empty.
func LoadProfiles(path string) ([]ScenarioEvalProfile, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return DefaultProfiles(), nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultProfiles(), nil
		}
		return nil, err
	}
	var file profilesFile
	if err := yaml.Unmarshal(raw, &file); err != nil {
		return nil, err
	}
	if len(file.Profiles) == 0 {
		return DefaultProfiles(), nil
	}
	return file.Profiles, nil
}

// ResolveProfilesPath picks config/scenario-eval-profiles.yaml when present.
func ResolveProfilesPath(roots ...string) string {
	candidates := []string{"config/scenario-eval-profiles.yaml"}
	for _, root := range roots {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		candidates = append(candidates, filepath.Join(root, "config", "scenario-eval-profiles.yaml"))
	}
	for _, p := range candidates {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}
	return ""
}

func profileForScenario(profiles []ScenarioEvalProfile, scenario string) ScenarioEvalProfile {
	scenario = strings.TrimSpace(scenario)
	for _, p := range profiles {
		if p.Scenario == scenario || p.ID == scenario {
			return p
		}
	}
	// Neutral weights when scenario is unknown.
	return ScenarioEvalProfile{
		ID: "default", Scenario: scenario, Label: scenario,
		Weights: map[string]float64{
			"quality": 0.25, "safety": 0.25, "efficiency": 0.25, "governance": 0.25,
		},
	}
}

func weightedRank(dims []DimensionScore, weights map[string]float64) float64 {
	var sumW, sum float64
	for _, d := range dims {
		w := weights[d.ID]
		if w <= 0 {
			continue
		}
		sumW += w
		sum += w * d.Score
	}
	if sumW == 0 {
		return 0
	}
	return sum / sumW
}
