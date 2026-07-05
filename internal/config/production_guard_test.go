package config

import (
	"strings"
	"testing"
)

func TestValidateProductionRejectsDevDefaults(t *testing.T) {
	cfg := Config{
		AuthMode:  "dev",
		JWTSecret: "dev-secret-change-me",
		SecretKey: "dev-secret-change-me",
	}
	issues := ValidateProduction(cfg)
	if len(issues) < 3 {
		t.Fatalf("issues=%v want >=3 production blockers", issues)
	}
}

func TestValidateProductionAcceptsStrongSecrets(t *testing.T) {
	cfg := Config{
		AuthMode:  "jwt",
		JWTSecret: "prod-jwt-secret-32chars-minimum!!",
		SecretKey: "prod-data-key-32chars-minimum!!!",
	}
	if issues := ValidateProduction(cfg); len(issues) != 0 {
		t.Fatalf("issues=%v want none", issues)
	}
}

func TestEnvFilePlaceholderIssues(t *testing.T) {
	lines := []string{
		"# comment",
		"ASH_DATABASE_URL=postgres://ash:CHANGE_ME_MIGRATOR_PW@host/ash",
		"ASH_JWT_SECRET=ok-strong-production-secret-value",
	}
	issues := EnvFilePlaceholderIssues(lines)
	if len(issues) != 1 || !strings.Contains(issues[0], "CHANGE_ME") {
		t.Fatalf("issues=%v want one CHANGE_ME line", issues)
	}
}

func TestProductionGuardSuite(t *testing.T) {
	t.Run("reject short jwt", func(t *testing.T) {
		cfg := Config{AuthMode: "jwt", JWTSecret: "short", SecretKey: "prod-data-key-32chars-minimum!!!"}
		if len(ValidateProduction(cfg)) == 0 {
			t.Fatal("expected jwt length issue")
		}
	})
}
