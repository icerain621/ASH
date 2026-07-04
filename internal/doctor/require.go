package doctor

import (
	"fmt"
	"strings"
)

// RequireCases fails when any listed case is missing, not pass, or skipped (optional).
func RequireCases(rep *Report, ids []string, rejectSkipped bool) error {
	if rep == nil {
		return fmt.Errorf("doctor report is nil")
	}
	byID := make(map[string]CaseResult, len(rep.Results))
	for _, r := range rep.Results {
		byID[r.ID] = r
	}
	var problems []string
	for _, id := range ids {
		r, ok := byID[id]
		if !ok {
			problems = append(problems, fmt.Sprintf("%s: missing from report", id))
			continue
		}
		if r.Status != "pass" {
			problems = append(problems, fmt.Sprintf("%s: status=%s %s", id, r.Status, r.Message))
			continue
		}
		if rejectSkipped && strings.Contains(strings.ToLower(r.Message), "skipped") {
			problems = append(problems, fmt.Sprintf("%s: skipped (%s)", id, r.Message))
			continue
		}
		if rejectSkipped {
			for _, ev := range r.Evidence {
				if ev.Kind == "skipped" {
					problems = append(problems, fmt.Sprintf("%s: evidence skipped (%s)", id, ev.Ref))
					break
				}
			}
		}
	}
	if len(problems) > 0 {
		return fmt.Errorf("doctor require failed:\n%s", strings.Join(problems, "\n"))
	}
	return nil
}
