package runs

import (
	"fmt"
	"strings"
	"time"

	"github.com/ash-repwiki/ash/internal/hooks"
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
	scenarioMin string,
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
		checks := s.runVerifyChecks(rec, runID, traceID, step, toolCtx, req, attempt, scenarioMin, stepRow, stepStart)
		if checks.waitErr != nil {
			return checks.waitErr
		}
		if checks.hookDeny {
			s.finishStep(stepRow, "failed", stepStart, errorCodeHookDenied, checks.detail)
			_, err := s.failRun(rec, runID, traceID, runStarted, errorCodeHookDenied, checks.detail)
			return err
		}
		if checks.ok {
			_, _ = s.eventsFor().Append(runID, traceID, "verify.passed", "info", map[string]any{
				"stepId": step.ID, "attempt": attempt,
			})
			return nil
		}
		lastErr = checks.detail
		_, _ = s.eventsFor().Append(runID, traceID, "verify.failed", "warn", map[string]any{
			"stepId": step.ID, "attempt": attempt, "error": checks.detail,
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

type verifyChecksResult struct {
	ok       bool
	detail   string
	hookDeny bool
	waitErr  error
}

func (s *Service) runVerifyChecks(
	rec *store.RunRecord,
	runID, traceID string,
	step rules.Step,
	toolCtx toolbus.Context,
	req CreateRequest,
	attempt int,
	scenarioMin string,
	stepRow *store.RunStep,
	stepStart time.Time,
) verifyChecksResult {
	for _, item := range step.Verify.Checks {
		if denied, reason := s.scenarioToolDenied(rec, item.Tool); denied {
			return verifyChecksResult{detail: reason}
		}
		risk := string(s.tools.ToolRisk(item.Tool))
		gate, msg, gateErr := s.applyPreToolUseGate(rec, runID, traceID, req.Inputs, step.ID, item, risk, stepRow, stepStart)
		if gateErr != nil {
			return verifyChecksResult{waitErr: gateErr}
		}
		if gate == preToolUseGateDeny {
			return verifyChecksResult{hookDeny: true, detail: msg}
		}
		if !s.dangerousToolAllowed(rec, req.Inputs, step.ID, item, risk) {
			return verifyChecksResult{detail: fmt.Sprintf("tool %s has danger risk and requires approval", item.Tool)}
		}
		res := s.callToolWithRetry(runID, traceID, step.ID, risk, rec.SpaceID, rec.PolicyProfile, scenarioMin, toolCtx, item, nil)
		if !res.OK {
			msg := res.Error
			if msg == "" {
				msg = "tool failed"
			}
			return verifyChecksResult{detail: fmt.Sprintf("%s: %s (attempt %d)", item.Tool, msg, attempt)}
		}
		postDec := s.evaluatePostToolUse(rec, step.ID, item.Tool, risk)
		if postDec.Action != hooks.ActionAllow || postDec.RuleIndex >= 0 {
			payload := hookPostDecisionPayload(step.ID, item.Tool, risk, postDec)
			_, _ = s.eventsFor().Append(runID, traceID, "hook.post_tool_use", "info", payload)
			_, _ = s.eventsFor().Append(runID, traceID, "hook.decision", "info", payload)
		}
		if postDec.Action == hooks.ActionDeny {
			msg := postDec.Reason
			if msg == "" {
				msg = fmt.Sprintf("PostToolUse hook denied tool %s", item.Tool)
			}
			return verifyChecksResult{hookDeny: true, detail: msg}
		}
	}
	return verifyChecksResult{ok: true}
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
