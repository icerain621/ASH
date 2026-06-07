package rules

import (
	_ "embed"

	"github.com/ash-repwiki/ash/internal/schemayaml"
)

//go:embed schemas/ash.rules.v0.1.schema.json
var rulesSchemaJSON []byte

// ValidateSchema checks raw YAML against the embedded ash.rules/v0.1 JSON Schema.
func ValidateSchema(raw []byte) []ValidationIssue {
	issues := schemayaml.Validate(rulesSchemaJSON, Version, raw)
	out := make([]ValidationIssue, len(issues))
	for i, item := range issues {
		out[i] = ValidationIssue(item)
	}
	return out
}
