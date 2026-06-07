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

// ParseAndValidate parses YAML, validates against the JSON Schema, then runs semantic checks.
func ParseAndValidate(raw []byte) ValidationResult {
	res := ValidationResult{OK: true}
	if issues := ValidateSchema(raw); len(issues) > 0 {
		res.OK = false
		res.Issues = append(res.Issues, issues...)
	}
	doc, err := ParseYAML(raw)
	if err != nil {
		res.OK = false
		res.Issues = append(res.Issues, ValidationIssue{
			Path:    "$",
			Code:    "YAML_PARSE_ERROR",
			Message: err.Error(),
		})
		return res
	}
	sem := Validate(doc)
	if !sem.OK {
		res.OK = false
		res.Issues = append(res.Issues, sem.Issues...)
	}
	res.Doc = doc
	return res
}
