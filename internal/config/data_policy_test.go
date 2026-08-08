package config

import "testing"

func TestLoadDataPolicyDefaults(t *testing.T) {
	t.Setenv("ASH_RETENTION_EVENTS_DAYS", "")
	t.Setenv("ASH_RETENTION_AUDIT_DAYS", "")
	t.Setenv("ASH_RETENTION_ARTIFACTS_DAYS", "")
	t.Setenv("ASH_RETENTION_ARTIFACTS_MAX_RUNS", "")
	t.Setenv("ASH_MEMORY_TTL_L1_DAYS", "")
	t.Setenv("ASH_MEMORY_TTL_L2_DAYS", "")
	t.Setenv("ASH_MEMORY_TTL_REVIEW_DAYS", "")

	p := LoadDataPolicy()
	if p.EventsDays != BuiltinRetentionEventsDays {
		t.Fatalf("events=%d want %d", p.EventsDays, BuiltinRetentionEventsDays)
	}
	if p.AuditDays != BuiltinRetentionAuditDays {
		t.Fatalf("audit=%d want %d", p.AuditDays, BuiltinRetentionAuditDays)
	}
	if p.ArtifactsDays != BuiltinRetentionArtifactsDays {
		t.Fatalf("artifactsDays=%d want %d", p.ArtifactsDays, BuiltinRetentionArtifactsDays)
	}
	if p.ArtifactsMaxRuns != BuiltinRetentionArtifactsMaxRuns {
		t.Fatalf("artifactsMaxRuns=%d want %d", p.ArtifactsMaxRuns, BuiltinRetentionArtifactsMaxRuns)
	}
	if p.MemoryTTLL1Days <= 0 || p.MemoryTTLL2Days <= 0 {
		t.Fatalf("memory ttl L1=%d L2=%d want positive", p.MemoryTTLL1Days, p.MemoryTTLL2Days)
	}
}

func TestRetentionEnvOverridesAndInvalidFallback(t *testing.T) {
	t.Setenv("ASH_RETENTION_EVENTS_DAYS", "14")
	t.Setenv("ASH_RETENTION_AUDIT_DAYS", "0")
	t.Setenv("ASH_RETENTION_ARTIFACTS_DAYS", "abc")
	t.Setenv("ASH_RETENTION_ARTIFACTS_MAX_RUNS", "50")

	if got := EffectiveRetentionEventsDays(); got != 14 {
		t.Fatalf("events=%d want 14", got)
	}
	if got := EffectiveRetentionAuditDays(); got != BuiltinRetentionAuditDays {
		t.Fatalf("audit=%d want fallback %d", got, BuiltinRetentionAuditDays)
	}
	if got := EffectiveRetentionArtifactsDays(); got != BuiltinRetentionArtifactsDays {
		t.Fatalf("artifactsDays=%d want fallback %d", got, BuiltinRetentionArtifactsDays)
	}
	if got := EffectiveRetentionArtifactsMaxRuns(); got != 50 {
		t.Fatalf("maxRuns=%d want 50", got)
	}
}

func TestValidSensitivity(t *testing.T) {
	for _, level := range []string{"normal", "restricted", "secret", "SECRET"} {
		if !ValidSensitivity(level) {
			t.Fatalf("want valid %q", level)
		}
	}
	if ValidSensitivity("confidential") {
		t.Fatal("confidential should be invalid")
	}
}
