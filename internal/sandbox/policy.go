package sandbox

import (
	"errors"
	"fmt"
	"strings"
)

// ErrPolicyDenied is returned when tool risk is incompatible with sandbox mode.
var ErrPolicyDenied = errors.New("sandbox policy denied")

// ForceIsolatedPolicy reports whether a scenario policyProfile requires ≥ isolated.
func ForceIsolatedPolicy(policyProfile string) bool {
	switch strings.ToLower(strings.TrimSpace(policyProfile)) {
	case "hotfix", "security":
		return true
	default:
		return false
	}
}

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
	switch normalizeMode(mode) {
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

func normalizeMode(mode string) string {
	m := strings.ToLower(strings.TrimSpace(mode))
	switch m {
	case ModeOff, ModeReadOnly, ModeWorkspaceWrite, ModeIsolated:
		return m
	default:
		return m
	}
}

// ModeAtLeast reports whether got is at least as strict as want.
func ModeAtLeast(got, want string) bool {
	return modeRank(got) >= modeRank(want)
}

// ResolveSandboxMode picks effective mode as max of profile, risk floor, and optional override.
// Override can only raise the floor (DX2); it cannot lower below risk requirements.
func ResolveSandboxMode(toolRisk, profileDefault, override string) string {
	return ResolveSandboxModeExt(toolRisk, profileDefault, override, "", "")
}

// ResolveSandboxModeExt merges scenario minMode and policyProfile force into the floor.
func ResolveSandboxModeExt(toolRisk, profileDefault, override, scenarioMin, policyProfile string) string {
	best := profileDefault
	if strings.TrimSpace(best) == "" {
		best = ModeOff
	}
	raise := func(m string) {
		m = normalizeMode(m)
		if m == "" {
			return
		}
		if modeRank(m) > modeRank(best) {
			best = m
		}
	}
	raise(MinModeForRisk(toolRisk))
	raise(scenarioMin)
	if ForceIsolatedPolicy(policyProfile) {
		raise(ModeIsolated)
	}
	raise(override)
	return normalizeMode(best)
}

// Authorize implements M4-SBX-02 / DX2: danger/network require ≥ isolated.
func Authorize(toolRisk, configuredMode string) error {
	mode := configuredMode
	if mode == "" {
		mode = ModeOff
	}
	min := MinModeForRisk(toolRisk)
	if modeRank(min) >= modeRank(ModeIsolated) && modeRank(mode) < modeRank(ModeIsolated) {
		return fmt.Errorf("%w: risk=%s requires mode>=%s got %s", ErrPolicyDenied, toolRisk, ModeIsolated, mode)
	}
	return nil
}
