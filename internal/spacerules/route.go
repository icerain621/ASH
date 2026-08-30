package spacerules

import (
	"strings"
)

// PickScenario applies document route keywords (preferScenario wins).
func PickScenario(goal string, doc Document) (name, reason string) {
	g := strings.ToLower(strings.TrimSpace(goal))
	if prefer := strings.TrimSpace(doc.PreferScenario); prefer != "" {
		return prefer, "space_rule:prefer:" + prefer
	}
	order := []string{"security_patch", "hotfix", "feature_delivery"}
	for _, scenario := range order {
		keys := doc.Route[scenario]
		for _, k := range keys {
			k = strings.TrimSpace(strings.ToLower(k))
			if k == "" {
				continue
			}
			if strings.Contains(g, k) {
				return scenario, "space_rule:" + scenario + ":" + k
			}
		}
	}
	return "feature_delivery", "space_rule:default"
}
