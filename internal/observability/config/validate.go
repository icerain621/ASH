package config

import (
	_ "embed"
	"fmt"
	"strings"

	"github.com/ash-repwiki/ash/internal/schemayaml"
	"gopkg.in/yaml.v3"
)

//go:embed schemas/ash.obs.v0.1.schema.json
var obsSchemaJSON []byte

// Default returns the M0-safe observability defaults (redaction on, outbound off).
func Default() Document {
	return Document{
		Version: Version,
		Redaction: RedactionConfig{
			Enabled: true,
			DenyKeys: []string{
				"password", "secret", "token", "authorization", "apiKey", "api_key",
			},
			MaskValue: "[REDACTED]",
		},
		Sampling: SamplingConfig{EventExportRate: 1},
		Export:   ExportConfig{AllowOutbound: false},
		Plugins: PluginsConfig{
			Prometheus: &PrometheusPlugin{Enabled: true, Path: "/metrics"},
			Console:    &ConsolePlugin{Enabled: true, Level: "info"},
			Otel:       &OtelPlugin{Enabled: false},
		},
	}
}

// ValidateSchema checks raw YAML against the embedded ash.obs/v0.1 JSON Schema.
func ValidateSchema(raw []byte) []ValidationIssue {
	return toIssues(schemayaml.Validate(obsSchemaJSON, Version, raw))
}

// ParseYAML parses YAML without validation.
func ParseYAML(raw []byte) (*Document, error) {
	var doc Document
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	return &doc, nil
}

// Validate applies semantic checks on a parsed document.
func Validate(doc *Document) ValidationResult {
	res := ValidationResult{OK: true, Doc: doc}
	if doc == nil {
		return fail(res, "$", "NULL_DOCUMENT", "document is nil")
	}
	if doc.Version != Version {
		res = fail(res, "$.version", "INVALID_VERSION",
			fmt.Sprintf("version must be %q, got %q", Version, doc.Version))
	}
	if doc.Export.AllowOutbound && !doc.Redaction.Enabled {
		res = fail(res, "$.export.allowOutbound", "REDACTION_REQUIRED",
			"outbound export requires redaction.enabled=true")
	}
	if doc.Plugins.Otel != nil && doc.Plugins.Otel.Enabled && strings.TrimSpace(doc.Plugins.Otel.Endpoint) == "" {
		res = fail(res, "$.plugins.otel.endpoint", "REQUIRED",
			"otel.endpoint is required when otel.enabled is true")
	}
	if doc.Sampling.EventExportRate < 0 || doc.Sampling.EventExportRate > 1 {
		res = fail(res, "$.sampling.eventExportRate", "OUT_OF_RANGE",
			"eventExportRate must be between 0 and 1")
	}
	return res
}

// ParseAndValidate parses YAML, validates schema, then runs semantic checks.
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
			Path: "$", Code: "YAML_PARSE_ERROR", Message: err.Error(),
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

func fail(res ValidationResult, path, code, msg string) ValidationResult {
	res.OK = false
	res.Issues = append(res.Issues, ValidationIssue{Path: path, Code: code, Message: msg})
	return res
}

func toIssues(in []schemayaml.Issue) []ValidationIssue {
	out := make([]ValidationIssue, len(in))
	for i, item := range in {
		out[i] = ValidationIssue(item)
	}
	return out
}
