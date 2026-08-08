package config

import (
	"os"
	"strconv"
	"strings"

	"github.com/ash-repwiki/ash/internal/memory"
)

// Data classification levels (appendix J / PRD §8).
const (
	SensitivityNormal     = "normal"
	SensitivityRestricted = "restricted"
	SensitivitySecret     = "secret"
)

// Built-in retention defaults when env overrides are unset (appendix J).
const (
	BuiltinRetentionEventsDays       = 90
	BuiltinRetentionAuditDays        = 365
	BuiltinRetentionArtifactsDays    = 30
	BuiltinRetentionArtifactsMaxRuns = 200
)

// DataPolicy is the effective org-wide classification + retention snapshot.
type DataPolicy struct {
	EventsDays          int `json:"eventsDays"`
	AuditDays           int `json:"auditDays"`
	ArtifactsDays       int `json:"artifactsDays"`
	ArtifactsMaxRuns    int `json:"artifactsMaxRuns"`
	MemoryTTLL1Days     int `json:"memoryTtlL1Days"`
	MemoryTTLL2Days     int `json:"memoryTtlL2Days"`
	MemoryTTLReviewDays int `json:"memoryTtlReviewDays"`
}

// LoadDataPolicy returns effective retention defaults (env overrides + memory TTL).
func LoadDataPolicy() DataPolicy {
	return DataPolicy{
		EventsDays:          EffectiveRetentionEventsDays(),
		AuditDays:           EffectiveRetentionAuditDays(),
		ArtifactsDays:       EffectiveRetentionArtifactsDays(),
		ArtifactsMaxRuns:    EffectiveRetentionArtifactsMaxRuns(),
		MemoryTTLL1Days:     memory.EffectiveTTLDaysL1(),
		MemoryTTLL2Days:     memory.EffectiveTTLDaysL2(),
		MemoryTTLReviewDays: memory.EffectiveTTLReviewLeadDays(),
	}
}

// EffectiveRetentionEventsDays returns ASH_RETENTION_EVENTS_DAYS (default 90).
func EffectiveRetentionEventsDays() int {
	return effectivePositiveInt("ASH_RETENTION_EVENTS_DAYS", BuiltinRetentionEventsDays)
}

// EffectiveRetentionAuditDays returns ASH_RETENTION_AUDIT_DAYS (default 365).
func EffectiveRetentionAuditDays() int {
	return effectivePositiveInt("ASH_RETENTION_AUDIT_DAYS", BuiltinRetentionAuditDays)
}

// EffectiveRetentionArtifactsDays returns ASH_RETENTION_ARTIFACTS_DAYS (default 30).
func EffectiveRetentionArtifactsDays() int {
	return effectivePositiveInt("ASH_RETENTION_ARTIFACTS_DAYS", BuiltinRetentionArtifactsDays)
}

// EffectiveRetentionArtifactsMaxRuns returns ASH_RETENTION_ARTIFACTS_MAX_RUNS (default 200).
func EffectiveRetentionArtifactsMaxRuns() int {
	return effectivePositiveInt("ASH_RETENTION_ARTIFACTS_MAX_RUNS", BuiltinRetentionArtifactsMaxRuns)
}

// ValidSensitivity reports whether level is one of normal|restricted|secret.
func ValidSensitivity(level string) bool {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case SensitivityNormal, SensitivityRestricted, SensitivitySecret:
		return true
	default:
		return false
	}
}

func effectivePositiveInt(envKey string, fallback int) int {
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
