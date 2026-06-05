package authz

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ash-repwiki/ash/internal/store"
)

// ToolRule defines scenario-scoped tool allow/deny for one actor role.
type ToolRule struct {
	Allow    []string `json:"allow,omitempty"`
	Deny     []string `json:"deny,omitempty"`
	DenyMode string   `json:"denyMode,omitempty"` // block | skip
}

// ScenarioToolPolicy is stored in resource_scopes.policyJson for resourceType=scenario.
type ScenarioToolPolicy struct {
	ToolMatrix map[string]ToolRule `json:"toolMatrix"`
}

// ScenarioMatrixRow is one scenario policy row for API responses.
type ScenarioMatrixRow struct {
	ScenarioKey string              `json:"scenarioKey"`
	Scenario    string              `json:"scenario"`
	Version     string              `json:"version"`
	ToolMatrix  map[string]ToolRule `json:"toolMatrix"`
}

func scenarioKey(name, version string) string {
	return name + "@" + version
}

func defaultScenarioPolicies() map[string]ScenarioToolPolicy {
	return map[string]ScenarioToolPolicy{
		scenarioKey("feature_delivery", "1.0.0"): {
			ToolMatrix: map[string]ToolRule{
				"maintainer": {Allow: []string{"*"}},
				"operator":   {Allow: []string{"git.*", "test.run", "apply_patch"}, Deny: []string{"runtime.command", "mcp.call"}},
				"reviewer":   {Allow: []string{"git.status", "git.diff", "test.run"}, Deny: []string{"apply_patch", "runtime.command", "mcp.call"}, DenyMode: "block"},
				"auditor":    {Allow: []string{"git.status"}, Deny: []string{"*"}, DenyMode: "block"},
			},
		},
		scenarioKey("hotfix", "1.0.0"): {
			ToolMatrix: map[string]ToolRule{
				"maintainer": {Allow: []string{"*"}},
				"operator":   {Allow: []string{"git.*", "test.run", "apply_patch"}, Deny: []string{"runtime.command"}},
				"reviewer":   {Allow: []string{"git.*", "test.run"}, Deny: []string{"runtime.command", "mcp.call"}, DenyMode: "block"},
			},
		},
		scenarioKey("security_patch", "1.0.0"): {
			ToolMatrix: map[string]ToolRule{
				"maintainer": {Allow: []string{"*"}},
				"operator":   {Allow: []string{"git.*", "test.run"}, Deny: []string{"apply_patch", "runtime.command", "mcp.call"}, DenyMode: "block"},
				"reviewer":   {Allow: []string{"git.status", "test.run"}, Deny: []string{"apply_patch", "runtime.command"}, DenyMode: "block"},
				"auditor":    {Allow: []string{"git.status"}, Deny: []string{"*"}, DenyMode: "block"},
			},
		},
	}
}

// DefaultScenarioPolicyJSON returns seeded policy JSON for a scenario.
func DefaultScenarioPolicyJSON(name, version string) string {
	policies := defaultScenarioPolicies()
	key := scenarioKey(name, version)
	policy, ok := policies[key]
	if !ok {
		policy = ScenarioToolPolicy{ToolMatrix: map[string]ToolRule{"maintainer": {Allow: []string{"*"}}}}
	}
	b, _ := json.Marshal(policy)
	return string(b)
}

// DefaultScenarioMatrix returns built-in scenario × role tool rules.
func DefaultScenarioMatrix() []ScenarioMatrixRow {
	policies := defaultScenarioPolicies()
	out := make([]ScenarioMatrixRow, 0, len(policies))
	for key, policy := range policies {
		name, version, _ := strings.Cut(key, "@")
		out = append(out, ScenarioMatrixRow{
			ScenarioKey: key, Scenario: name, Version: version, ToolMatrix: policy.ToolMatrix,
		})
	}
	return out
}

// ParseScenarioToolPolicy parses policy JSON from a resource scope row.
func ParseScenarioToolPolicy(raw string) (ScenarioToolPolicy, error) {
	var policy ScenarioToolPolicy
	if strings.TrimSpace(raw) == "" {
		return policy, nil
	}
	if err := json.Unmarshal([]byte(raw), &policy); err != nil {
		return policy, err
	}
	if policy.ToolMatrix == nil {
		policy.ToolMatrix = map[string]ToolRule{}
	}
	return policy, nil
}

// EvaluateScenarioTool checks role/tool against a scenario policy JSON blob.
func EvaluateScenarioTool(policyJSON, actorRole, tool string) (allowed bool, reason string) {
	policy, err := ParseScenarioToolPolicy(policyJSON)
	if err != nil || len(policy.ToolMatrix) == 0 {
		return true, ""
	}
	role := firstNonEmpty(actorRole, "maintainer")
	rule, ok := policy.ToolMatrix[role]
	if !ok {
		rule = policy.ToolMatrix["maintainer"]
	}
	for _, pattern := range rule.Deny {
		if toolPatternMatch(pattern, tool) {
			mode := firstNonEmpty(rule.DenyMode, "block")
			return false, fmt.Sprintf("scenario tool denied for role %s: %s (%s)", role, tool, mode)
		}
	}
	if len(rule.Allow) == 0 {
		return true, ""
	}
	for _, pattern := range rule.Allow {
		if toolPatternMatch(pattern, tool) {
			return true, ""
		}
	}
	return false, fmt.Sprintf("scenario tool not in allow list for role %s: %s", role, tool)
}

// LoadScenarioPolicy loads scenario tool policy from resource_scopes.
func SeedScenarioScopes(db *store.DB, spaceID string, now time.Time) error {
	if db == nil {
		return nil
	}
	return seedScenarioScopes(db.DB, spaceID, now)
}

// SeedScenarioScopesTx seeds scenario policies inside an existing SQL transaction.
func SeedScenarioScopesTx(tx *gorm.DB, spaceID string, now time.Time) error {
	if tx == nil {
		return nil
	}
	return seedScenarioScopes(tx, spaceID, now)
}

func seedScenarioScopes(db *gorm.DB, spaceID string, now time.Time) error {
	scenarios := []struct{ name, version string }{
		{name: "feature_delivery", version: "1.0.0"},
		{name: "hotfix", version: "1.0.0"},
		{name: "security_patch", version: "1.0.0"},
	}
	for _, item := range scenarios {
		key := scenarioKey(item.name, item.version)
		var count int64
		if err := db.Model(&store.ResourceScope{}).
			Where("space_id = ? AND resource_type = ? AND resource_id = ?", spaceID, "scenario", key).
			Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			continue
		}
		row := store.ResourceScope{
			ID: "scope_" + uuid.NewString(), SpaceID: spaceID,
			ResourceType: "scenario", ResourceID: key,
			PolicyJSON: DefaultScenarioPolicyJSON(item.name, item.version),
			CreatedAt: now, UpdatedAt: now,
		}
		if err := db.Create(&row).Error; err != nil {
			return err
		}
	}
	return nil
}

func LoadScenarioPolicy(db *store.DB, spaceID, scenarioName, scenarioVersion string) (string, error) {
	if db == nil {
		return DefaultScenarioPolicyJSON(scenarioName, scenarioVersion), nil
	}
	key := scenarioKey(scenarioName, scenarioVersion)
	var row store.ResourceScope
	err := db.Where(
		"space_id = ? AND resource_type = ? AND resource_id = ?",
		firstNonEmpty(spaceID, "local"), "scenario", key,
	).First(&row).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return DefaultScenarioPolicyJSON(scenarioName, scenarioVersion), nil
		}
		return "", err
	}
	return row.PolicyJSON, nil
}

// ScenarioPoliciesForSpace returns scenario tool matrix rows stored for a space.
func ScenarioPoliciesForSpace(db *store.DB, spaceID string) ([]ScenarioMatrixRow, error) {
	var rows []store.ResourceScope
	if err := db.Where("space_id = ? AND resource_type = ?", spaceID, "scenario").
		Order("resource_id asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]ScenarioMatrixRow, 0, len(rows))
	for _, row := range rows {
		policy, err := ParseScenarioToolPolicy(row.PolicyJSON)
		if err != nil {
			return nil, err
		}
		name, version, _ := strings.Cut(row.ResourceID, "@")
		out = append(out, ScenarioMatrixRow{
			ScenarioKey: row.ResourceID, Scenario: name, Version: version, ToolMatrix: policy.ToolMatrix,
		})
	}
	if len(out) == 0 {
		return DefaultScenarioMatrix(), nil
	}
	return out, nil
}

func toolPatternMatch(pattern, tool string) bool {
	pattern = strings.TrimSpace(pattern)
	tool = strings.TrimSpace(tool)
	if pattern == "" {
		return false
	}
	if pattern == "*" {
		return true
	}
	if strings.HasSuffix(pattern, ".*") {
		return strings.HasPrefix(tool, strings.TrimSuffix(pattern, ".*"))
	}
	return pattern == tool
}

func permissionMatch(grant, want string) bool {
	grant = strings.TrimSpace(grant)
	want = strings.TrimSpace(want)
	if grant == "" || want == "" {
		return false
	}
	if grant == "*" || grant == want {
		return true
	}
	if strings.HasSuffix(grant, ":*") {
		return strings.HasPrefix(want, strings.TrimSuffix(grant, "*"))
	}
	return false
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
