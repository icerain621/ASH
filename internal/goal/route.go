package goal

import (
	"fmt"
	"strings"

	"github.com/ash-repwiki/ash/internal/rules"
	"github.com/ash-repwiki/ash/internal/spacerules"
)

// RouteResult is the scenario chosen for a natural-language goal.
type RouteResult struct {
	ScenarioName    string
	ScenarioVersion string
	Reason          string
	PolicyProfile   string
	Steps           []StepPreview
	Inputs          map[string]any
}

// StepPreview is a lightweight plan step for UI / CLI.
type StepPreview struct {
	ID   string `json:"id"`
	Role string `json:"role"`
	Kind string `json:"kind"`
}

// Route maps goal text using builtin Space Rules keywords (DJ compatible).
func Route(goal string, scenarios *rules.Loader, repoRoot string) (*RouteResult, error) {
	return RouteWithDoc(goal, scenarios, repoRoot, spacerules.BuiltinDocument())
}

// RouteWithDoc applies a Space Rules document for scenario pick and input defaults.
func RouteWithDoc(goal string, scenarios *rules.Loader, repoRoot string, rulesDoc spacerules.Document) (*RouteResult, error) {
	goal = strings.TrimSpace(goal)
	if goal == "" {
		return nil, fmt.Errorf("goal is required")
	}
	if scenarios == nil {
		return nil, fmt.Errorf("scenario loader required")
	}
	rulesDoc = spacerules.NormalizeDocument(rulesDoc)
	name, reason := spacerules.PickScenario(goal, rulesDoc)
	scenDoc, err := latestDoc(scenarios, name)
	if err != nil {
		return nil, err
	}
	root := strings.TrimSpace(repoRoot)
	if root == "" {
		root = "."
	}
	steps := make([]StepPreview, 0, len(scenDoc.Scenario.Steps))
	for _, st := range scenDoc.Scenario.Steps {
		steps = append(steps, StepPreview{ID: st.ID, Role: st.Role, Kind: st.Kind})
	}
	policy := strings.TrimSpace(rulesDoc.Defaults.PolicyProfile)
	if policy == "" {
		policy = scenDoc.Scenario.PolicyProfile
	}
	if policy == "" {
		policy = "default"
	}
	inputs := map[string]any{
		"issueOrSpec": goal,
		"repoRoot":    root,
	}
	for k, v := range rulesDoc.Defaults.Inputs {
		if _, exists := inputs[k]; !exists {
			inputs[k] = v
		}
	}
	return &RouteResult{
		ScenarioName:    scenDoc.Scenario.Name,
		ScenarioVersion: scenDoc.Scenario.ScenarioVersion,
		Reason:          reason,
		PolicyProfile:   policy,
		Steps:           steps,
		Inputs:          inputs,
	}, nil
}

func latestDoc(loader *rules.Loader, name string) (*rules.Document, error) {
	var best *rules.Document
	var bestVer string
	for _, sum := range loader.List() {
		if sum.Name != name {
			continue
		}
		d, err := loader.Get(sum.Name, sum.ScenarioVersion)
		if err != nil {
			continue
		}
		if best == nil || sum.ScenarioVersion > bestVer {
			best = d
			bestVer = sum.ScenarioVersion
		}
	}
	if best == nil {
		return nil, fmt.Errorf("scenario %q not loaded", name)
	}
	return best, nil
}

func pickScenario(goal string) (name, reason string) {
	name, reason = spacerules.PickScenario(goal, spacerules.BuiltinDocument())
	// Preserve DJ-era reason prefix for builtin-only callers/tests.
	if strings.HasPrefix(reason, "space_rule:") {
		if strings.HasSuffix(reason, ":default") || strings.Contains(reason, ":prefer:") {
			return name, strings.Replace(reason, "space_rule:", "keyword:", 1)
		}
		// space_rule:security_patch:cve -> keyword:cve
		parts := strings.Split(reason, ":")
		if len(parts) >= 3 {
			return name, "keyword:" + parts[len(parts)-1]
		}
	}
	return name, reason
}
