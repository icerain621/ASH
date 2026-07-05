package config

import (
	"strings"
)

// Weak production secrets and template placeholders that must be rotated before deploy.
var weakSecrets = []string{
	"dev-secret-change-me",
	"change-me",
	"secret",
	"password",
	"ash",
	"ash_app",
}

// ProductionIssue describes a blocking production configuration problem.
type ProductionIssue struct {
	Field   string
	Message string
}

// ValidateProduction returns issues when cfg is unsafe for a production Worker.
func ValidateProduction(cfg Config) []ProductionIssue {
	var issues []ProductionIssue
	if mode := strings.ToLower(strings.TrimSpace(cfg.AuthMode)); mode == "dev" || mode == "disabled" || mode == "" {
		issues = append(issues, ProductionIssue{
			Field:   "ASH_AUTH_MODE",
			Message: "must be jwt (not dev/disabled) in production",
		})
	}
	if weak, why := secretWeakness(cfg.JWTSecret); weak {
		issues = append(issues, ProductionIssue{Field: "ASH_JWT_SECRET", Message: why})
	}
	if weak, why := secretWeakness(cfg.SecretKey); weak {
		issues = append(issues, ProductionIssue{Field: "ASH_SECRET_KEY", Message: why})
	}
	return issues
}

func secretWeakness(value string) (bool, string) {
	v := strings.TrimSpace(value)
	if v == "" {
		return true, "must be set"
	}
	if len(v) < 16 {
		return true, "must be at least 16 characters"
	}
	upper := strings.ToUpper(v)
	if strings.Contains(upper, "CHANGE_ME") {
		return true, "contains template placeholder"
	}
	for _, weak := range weakSecrets {
		if strings.EqualFold(v, weak) {
			return true, "uses dev/default value"
		}
	}
	return false, ""
}

// EnvFilePlaceholderIssues scans key=value env files for unresolved template markers.
func EnvFilePlaceholderIssues(lines []string) []string {
	var issues []string
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		if trim == "" || strings.HasPrefix(trim, "#") {
			continue
		}
		if strings.Contains(trim, "CHANGE_ME") {
			issues = append(issues, trim)
		}
	}
	return issues
}
