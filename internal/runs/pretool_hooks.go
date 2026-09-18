package runs

import (
	"fmt"
	"strings"

	"github.com/ash-repwiki/ash/internal/hooks"
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
