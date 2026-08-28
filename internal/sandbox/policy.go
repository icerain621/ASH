package sandbox

import (
	"errors"
	"fmt"
	"strings"
)

// ErrPolicyDenied is returned when tool risk is incompatible with sandbox mode.
var ErrPolicyDenied = errors.New("sandbox policy denied")

// MinModeForRisk returns the lowest allowed sandbox mode for a tool risk class.
func MinModeForRisk(risk string) string {
	switch strings.ToLower(strings.TrimSpace(risk)) {
	case "danger", "network":
		return ModeIsolated
	case "medium", "write":
		return ModeWorkspaceWrite
	default:
		return ModeOff
	}
}

func modeRank(mode string) int {
	switch mode {
	case ModeOff:
		return 0
	case ModeReadOnly:
		return 1
	case ModeWorkspaceWrite:
		return 2
	case ModeIsolated:
		return 3
	default:
		return 0
	}
}

// ResolveSandboxMode picks effective mode: override > max(profile, risk floor) > off.
func ResolveSandboxMode(toolRisk, profileDefault, override string) string {
	if override != "" {
		return override
	}
	base := profileDefault
	if base == "" {
		base = ModeOff
	}
	floor := MinModeForRisk(toolRisk)
	if modeRank(floor) > modeRank(base) {
		return floor
	}
	return base
}

// Authorize implements M4-SBX-02: danger/network tools cannot run under sandboxMode=off.
// Stricter floors are applied by ResolveSandboxMode when selecting an executor.
func Authorize(toolRisk, configuredMode string) error {
	mode := configuredMode
	if mode == "" {
		mode = ModeOff
	}
	min := MinModeForRisk(toolRisk)
	if min == ModeIsolated && mode == ModeOff {
		return fmt.Errorf("%w: risk=%s requires mode>=%s got %s", ErrPolicyDenied, toolRisk, min, mode)
	}
	return nil
}
