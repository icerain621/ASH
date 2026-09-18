package runs

import (
	"fmt"
	"strings"
	"time"

	"github.com/ash-repwiki/ash/internal/hooks"
	"github.com/ash-repwiki/ash/internal/rules"
	"github.com/ash-repwiki/ash/internal/store"
)

const (
	errorCodeHookDenied              = "HOOK_DENIED"
	errorCodeHookAskApprovalRequired = "HOOK_ASK_APPROVAL_REQUIRED"
	gateHookPreToolUse               = "hook_pre_tool_use"
	approvedHookPreToolUseStepsKey   = "_approvedHookPreToolUseSteps"
)

func (s *Service) loadSpaceHooksConfig(spaceID string) (hooks.Config, error) {
	if s == nil {
		return hooks.Config{}, nil
	}
	spaceID = firstNonEmpty(strings.TrimSpace(spaceID), "local")
	var row store.SpacePolicyPack
	if err := s.gdb().First(&row, "space_id = ?", spaceID).Error; err != nil {
		return hooks.Config{}, nil
	}
	return hooks.ConfigFromSpaceBodyJSON(row.BodyJSON)
}

// evaluatePreToolUse loads space hooks and evaluates PreToolUse.
// No SpacePolicy / empty hooks → allow (unchanged behavior).
// Load/parse errors fail-closed as deny. Prior hook-ask approval for the step → allow.
func (s *Service) evaluatePreToolUse(rec *store.RunRecord, inputs map[string]any, stepID, tool, risk string) hooks.Decision {
	if approvedStep(inputs, approvedHookPreToolUseStepsKey, stepID) {
		return hooks.Decision{Action: hooks.ActionAllow, Reason: "hook ask previously approved", RuleIndex: -1}
	}
	spaceID := ""
	runID := ""
	if rec != nil {
		spaceID = rec.SpaceID
		runID = rec.ID
	}
	cfg, err := s.loadSpaceHooksConfig(spaceID)
	if err != nil {
		return hooks.Decision{
			Action:    hooks.ActionDeny,
			Reason:    fmt.Sprintf("hooks config load failed: %v", err),
			RuleIndex: -1,
		}
	}
	return hooks.Evaluate(cfg, hooks.EventPreToolUse, hooks.ToolContext{
		Tool: tool, Risk: risk, SpaceID: firstNonEmpty(spaceID, "local"),
		RunID: runID, StepID: stepID,
	})
}

func hookDecisionPayload(stepID, tool, risk string, dec hooks.Decision) map[string]any {
	return map[string]any{
		"stepId":    stepID,
		"tool":      tool,
		"risk":      risk,
		"event":     string(hooks.EventPreToolUse),
		"action":    string(dec.Action),
		"reason":    dec.Reason,
		"ruleIndex": dec.RuleIndex,
	}
}

type preToolUseGateOutcome int

const (
	preToolUseGateAllow preToolUseGateOutcome = iota
	preToolUseGateDeny
)

// applyPreToolUseGate evaluates PreToolUse, emits hook events, and for ask enters waiting_approval
// (same gate as tool_chain). Deny returns preToolUseGateDeny and a message; allow returns preToolUseGateAllow.
// Ask returns ErrWaitingApproval after persisting approval state.
func (s *Service) applyPreToolUseGate(
	rec *store.RunRecord,
	runID, traceID string,
	inputs map[string]any,
	stepID string,
	item rules.ToolChainItem,
	risk string,
	stepRow *store.RunStep,
	stepStart time.Time,
) (preToolUseGateOutcome, string, error) {
	hookDec := s.evaluatePreToolUse(rec, inputs, stepID, item.Tool, risk)
	if hookDec.Action != hooks.ActionAllow || hookDec.RuleIndex >= 0 {
		payload := hookDecisionPayload(stepID, item.Tool, risk, hookDec)
		_, _ = s.eventsFor().Append(runID, traceID, "hook.pre_tool_use", "info", payload)
		_, _ = s.eventsFor().Append(runID, traceID, "hook.decision", "info", payload)
	}
	switch hookDec.Action {
	case hooks.ActionDeny:
		msg := hookDec.Reason
		if msg == "" {
			msg = fmt.Sprintf("PreToolUse hook denied tool %s", item.Tool)
		}
		_, _ = s.eventsFor().Append(runID, traceID, "policy.denied", "warn", map[string]any{
			"target": "tool", "reason": msg, "action": "deny", "ref": item.Tool,
			"matrix": "hooks.pre_tool_use",
		})
		return preToolUseGateDeny, msg, nil
	case hooks.ActionAsk:
		msg := hookDec.Reason
		if msg == "" {
			msg = fmt.Sprintf("PreToolUse hook requires approval for tool %s", item.Tool)
		}
		if setErr := s.trySetRunStatus(rec, StatusWaitingApproval); setErr != nil {
			return preToolUseGateDeny, "", setErr
		}
		rec.UpdatedAt = time.Now().UTC()
		_ = s.gdb().Save(rec).Error
		s.finishStep(stepRow, "waiting_approval", stepStart, errorCodeHookAskApprovalRequired, msg)
		_, _ = s.eventsFor().Append(runID, traceID, "gate.waiting_approval", "warn", map[string]any{
			"stepId": stepID, "gate": gateHookPreToolUse, "tool": item.Tool, "risk": risk, "reason": msg,
		})
		s.requestApproval(rec, stepRow, gateHookPreToolUse, risk, msg, map[string]any{
			"stepId": stepID, "tool": item.Tool, "policy": item.Policy,
		})
		return preToolUseGateDeny, msg, ErrWaitingApproval
	default:
		return preToolUseGateAllow, "", nil
	}
}
