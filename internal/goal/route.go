package goal

import (
	"fmt"
	"strings"

	"github.com/ash-repwiki/ash/internal/rules"
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

// Route maps goal text to one of the three delivery scenarios via keywords.
func Route(goal string, scenarios *rules.Loader, repoRoot string) (*RouteResult, error) {
	goal = strings.TrimSpace(goal)
	if goal == "" {
		return nil, fmt.Errorf("goal is required")
	}
	if scenarios == nil {
		return nil, fmt.Errorf("scenario loader required")
	}
	name, reason := pickScenario(goal)
	doc, err := latestDoc(scenarios, name)
	if err != nil {
		return nil, err
	}
	root := strings.TrimSpace(repoRoot)
	if root == "" {
		root = "."
	}
	steps := make([]StepPreview, 0, len(doc.Scenario.Steps))
	for _, st := range doc.Scenario.Steps {
		steps = append(steps, StepPreview{ID: st.ID, Role: st.Role, Kind: st.Kind})
	}
	policy := doc.Scenario.PolicyProfile
	if policy == "" {
		policy = "default"
	}
	return &RouteResult{
		ScenarioName:    doc.Scenario.Name,
		ScenarioVersion: doc.Scenario.ScenarioVersion,
		Reason:          reason,
		PolicyProfile:   policy,
		Steps:           steps,
		Inputs: map[string]any{
			"issueOrSpec": goal,
			"repoRoot":    root,
		},
	}, nil
}

func pickScenario(goal string) (name, reason string) {
	g := strings.ToLower(goal)
	securityKeys := []string{"security", "cve", "vuln", "漏洞", "安全", "xss", "rce"}
	for _, k := range securityKeys {
		if strings.Contains(g, k) {
			return "security_patch", "keyword:" + k
		}
	}
	hotfixKeys := []string{"hotfix", "urgent", "prod outage", "线上", "热修", "生产故障", "p0"}
	for _, k := range hotfixKeys {
		if strings.Contains(g, k) {
			return "hotfix", "keyword:" + k
		}
	}
	return "feature_delivery", "default"
}

func latestDoc(loader *rules.Loader, name string) (*rules.Document, error) {
	var best *rules.Document
	var bestVer string
	for _, sum := range loader.List() {
		if sum.Name != name {
			continue
		}
		doc, err := loader.Get(sum.Name, sum.ScenarioVersion)
		if err != nil {
			continue
		}
		if best == nil || sum.ScenarioVersion > bestVer {
			best = doc
			bestVer = sum.ScenarioVersion
		}
	}
	if best == nil {
		return nil, fmt.Errorf("scenario %q not loaded", name)
	}
	return best, nil
}
