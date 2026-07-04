package memory

import (
	"os"
	"strconv"
	"strings"
)

// Built-in default TTL when env overrides are unset (appendix C §4).
const (
	BuiltinTTLDaysL1 = 90
	BuiltinTTLDaysL2 = 365
)

// EffectiveTTLDaysL1 returns L1 default ttl days (ASH_MEMORY_TTL_L1_DAYS overrides builtin).
func EffectiveTTLDaysL1() int {
	return effectiveTTLDays("ASH_MEMORY_TTL_L1_DAYS", BuiltinTTLDaysL1)
}

// EffectiveTTLDaysL2 returns L2 default ttl days (ASH_MEMORY_TTL_L2_DAYS overrides builtin).
func EffectiveTTLDaysL2() int {
	return effectiveTTLDays("ASH_MEMORY_TTL_L2_DAYS", BuiltinTTLDaysL2)
}

func effectiveTTLDays(envKey string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(envKey))
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}
