package rules

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// ParseYAML parses raw YAML into a Document without semantic validation.
func ParseYAML(raw []byte) (*Document, error) {
	var doc Document
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("yaml parse: %w", err)
	}
	return &doc, nil
}

// ParseAndValidate parses YAML and runs semantic validation.
func ParseAndValidate(raw []byte) ValidationResult {
	doc, err := ParseYAML(raw)
	if err != nil {
		return ValidationResult{
			OK: false,
			Issues: []ValidationIssue{{
				Path:    "$",
				Code:    "YAML_PARSE_ERROR",
				Message: err.Error(),
			}},
		}
	}
	return Validate(doc)
}
