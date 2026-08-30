package runs

import (
	"fmt"
	"strings"
	"time"

	"github.com/ash-repwiki/ash/internal/rules"
	"github.com/ash-repwiki/ash/internal/store"
	"github.com/ash-repwiki/ash/internal/toolbus"
)

func verifyMaxAttempts(step rules.Step) int {
	if step.Retry != nil && step.Retry.MaxAttempts > 0 {
		return step.Retry.MaxAttempts
	}
	return 1
}

func verifyBackoff(step rules.Step) time.Duration {
	if step.Retry != nil && step.Retry.BackoffMs > 0 {
		return time.Duration(step.Retry.BackoffMs) * time.Millisecond
	}
	return 0
}

func (s *Service) executeVerifyStep(
	rec *store.RunRecord,
	runID, traceID string,
	runStarted, stepStart time.Time,
	stepRow *store.RunStep,
	step rules.Step,
	toolCtx toolbus.Context,
	req CreateRequest,
) error {
	if step.Verify == nil || len(step.Verify.Checks) == 0 {
		msg := "verify step missing checks"
		s.finishStep(stepRow, "failed", stepStart, "VERIFY_INVALID", msg)
		_, err := s.failRun(rec, runID, traceID, runStarted, "VERIFY_INVALID", msg)
		return err
	}
	attempts := verifyMaxAttempts(step)
	var lastErr string
	for attempt := 1; attempt <= attempts; attempt++ {
		_, _ = s.eventsFor().Append(runID, traceID, "verify.attempt", "info", map[string]any{
			"stepId": step.ID, "attempt": attempt, "maxAttempts": attempts,
		})
		ok, detail := s.runVerifyChecks(rec, runID, traceID, step, toolCtx, req, attempt)
		if ok {
			_, _ = s.eventsFor().Append(runID, traceID, "verify.passed", "info", map[string]any{
				"stepId": step.ID, "attempt": attempt,
			})
			return nil
		}
		lastErr = detail
		_, _ = s.eventsFor().Append(runID, traceID, "verify.failed", "warn", map[string]any{
			"stepId": step.ID, "attempt": attempt, "error": detail,
		})
		if attempt < attempts {
			if d := verifyBackoff(step); d > 0 {
				time.Sleep(d)
			}
		}
	}

	onFail := strings.TrimSpace(step.Verify.OnFail)
	if onFail == "" {
		onFail = "fail"
	}
	s.finishStep(stepRow, "failed", stepStart, "VERIFY_FAILED", lastErr)
	_, err := s.failRun(rec, runID, traceID, runStarted, "VERIFY_FAILED", lastErr)
	if onFail == "improve" {
		s.maybeDraftImprove(rec, runID, step.ID, lastErr)
	}
	return err
}

func (s *Service) runVerifyChecks(
	rec *store.RunRecord,
	runID, traceID string,
	step rules.Step,
	toolCtx toolbus.Context,
	req CreateRequest,
	attempt int,
) (bool, string) {
	for _, item := range step.Verify.Checks {
		if denied, reason := s.scenarioToolDenied(rec, item.Tool); denied {
			return false, reason
		}
		risk := string(s.tools.ToolRisk(item.Tool))
		if !s.dangerousToolAllowed(req.Inputs, step.ID, item, risk) {
			return false, fmt.Sprintf("tool %s has danger risk and requires approval", item.Tool)
		}
		res := s.callToolWithRetry(runID, traceID, step.ID, risk, rec.SpaceID, toolCtx, item)
		if !res.OK {
			msg := res.Error
			if msg == "" {
				msg = "tool failed"
			}
			return false, fmt.Sprintf("%s: %s (attempt %d)", item.Tool, msg, attempt)
		}
	}
	return true, ""
}

func (s *Service) maybeDraftImprove(rec *store.RunRecord, runID, stepID, detail string) {
	if s == nil || s.improve == nil || rec == nil {
		return
	}
	proposalID, err := s.improve.DraftFromVerifyFailure(rec.SpaceID, runID, stepID, detail)
	if err != nil {
		if s.eventsFor() != nil {
			_, _ = s.eventsFor().Append(runID, rec.TraceID, "improve.draft_failed", "warn", map[string]any{
				"stepId": stepID, "error": err.Error(),
			})
		}
		return
	}
	if s.eventsFor() != nil {
		_, _ = s.eventsFor().Append(runID, rec.TraceID, "improve.draft_created", "info", map[string]any{
			"stepId": stepID, "proposalId": proposalID, "reason": "verify_failed",
		})
	}
}
