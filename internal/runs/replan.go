package runs

import (
	"fmt"
	"math"
	"strings"

	"github.com/ash-repwiki/ash/internal/harness/drive"
	"github.com/ash-repwiki/ash/internal/review"
	"github.com/ash-repwiki/ash/internal/rules"
	"github.com/ash-repwiki/ash/internal/store"
)

// processReplan is the in-process review queue for harness replan proposals (F06).
var processReplan = review.NewService()

func replanMaxTurns(step rules.Step) int {
	if step.Verify != nil && step.Verify.MaxReplanTurns > 0 {
		return step.Verify.MaxReplanTurns
	}
	return drive.DefaultMaxReplanTurns
}

func replanStepIDFragment(stepID string) string {
	// Compact JSON fragment from encoding/json; exact field match avoids substring collisions.
	return `%"stepId":"` + stepID + `"%`
}

func (s *Service) countReplanProposals(runID, stepID string) (int, error) {
	if s == nil || s.db == nil || strings.TrimSpace(runID) == "" || strings.TrimSpace(stepID) == "" {
		return 0, fmt.Errorf("replan count unavailable")
	}
	var n int64
	err := s.db.Model(&store.RunEvent{}).
		Where("run_id = ? AND type = ? AND payload_json LIKE ?", runID, "harness.replan_proposed", replanStepIDFragment(stepID)).
		Count(&n).Error
	if err != nil {
		return 0, err
	}
	return int(n), nil
}

// maybeProposeReplan emits a bounded replan proposal into review (+ optional improve draft).
// Returns exhausted=true when the budget is already spent or the bound cannot be enforced.
func (s *Service) maybeProposeReplan(rec *store.RunRecord, runID string, step rules.Step, detail string) (exhausted bool) {
	if s == nil || rec == nil {
		return true
	}
	maxTurns := replanMaxTurns(step)
	ev := s.eventsFor()
	if ev == nil {
		return true
	}

	proposed, err := s.countReplanProposals(runID, step.ID)
	if err != nil {
		_, _ = ev.Append(runID, rec.TraceID, "harness.replan_exhausted", "warn", map[string]any{
			"stepId": step.ID, "maxReplanTurns": maxTurns, "proposed": math.MaxInt32, "error": err.Error(),
		})
		return true
	}
	remaining, ok := drive.ReplanBudget(proposed, maxTurns)
	if !ok {
		_, _ = ev.Append(runID, rec.TraceID, "harness.replan_exhausted", "warn", map[string]any{
			"stepId": step.ID, "maxReplanTurns": maxTurns, "proposed": proposed,
		})
		return true
	}

	// Budget token first: without a durable proposed event the bound cannot be counted.
	_, aerr := ev.Append(runID, rec.TraceID, "harness.replan_proposed", "info", map[string]any{
		"component":      "harness",
		"stepId":         step.ID,
		"maxReplanTurns": maxTurns,
		"proposed":       proposed + 1,
		"remaining":      remaining - 1,
		"detail":         detail,
		"awaitingReview": true,
	})
	if aerr != nil {
		_, _ = ev.Append(runID, rec.TraceID, "harness.replan_exhausted", "warn", map[string]any{
			"stepId": step.ID, "maxReplanTurns": maxTurns, "proposed": proposed, "error": aerr.Error(),
		})
		return true
	}

	target := fmt.Sprintf("%s:%s", runID, step.ID)
	reason := fmt.Sprintf("verify failed; replan remaining=%d/%d: %s", remaining, maxTurns, detail)
	if _, err := processReplan.Submit(review.KindRunSample, target, reason); err == nil && s.db != nil {
		_, _ = s.db.CreateReviewItem(rec.SpaceID, string(review.KindRunSample), target, reason)
	}
	if s.improve != nil {
		_, _ = s.improve.DraftFromVerifyFailure(rec.SpaceID, runID, step.ID, "replan: "+detail)
	}
	return false
}
