package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestDefault_isValid(t *testing.T) {
	doc := Default()
	raw, err := yamlMarshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	res := ParseAndValidate(raw)
	if !res.OK {
		t.Fatalf("default invalid: %+v", res.Issues)
	}
}

func TestLoad_usesRepoDefaultFile(t *testing.T) {
	if _, err := os.Stat(filepath.Join("..", "..", "..", "config", "ash-observability.yaml")); err != nil {
		t.Skip("repo config not found from test cwd")
	}
	t.Chdir(filepath.Join("..", "..", ".."))
	doc, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !doc.Redaction.Enabled {
		t.Fatal("expected redaction enabled")
	}
	if doc.Export.AllowOutbound {
		t.Fatal("expected outbound export disabled")
	}
}

func TestValidateSchema_invalidSamples(t *testing.T) {
	cases := []struct {
		name string
		yaml string
		want string
	}{
		{"wrong_version", `version: "ash.obs/v0.0"`, "version"},
		{"missing_version", `export: {allowOutbound: false}`, "version"},
		{"unknown_top_level", `version: "ash.obs/v0.1"
extra: true`, "additional"},
		{"invalid_sampling", `version: "ash.obs/v0.1"
sampling: {eventExportRate: 1.5}`, "eventExportRate"},
		{"invalid_console_level", `version: "ash.obs/v0.1"
plugins:
  console: {enabled: true, level: trace}`, "level"},
		{"prometheus_bad_path", `version: "ash.obs/v0.1"
plugins:
  prometheus: {enabled: true, path: metrics}`, "path"},
		{"otel_extra_field", `version: "ash.obs/v0.1"
plugins:
  otel: {enabled: false, secret: leak}`, "additional"},
		{"redaction_extra_field", `version: "ash.obs/v0.1"
redaction: {enabled: true, password: x}`, "additional"},
		{"empty_deny_key", `version: "ash.obs/v0.1"
redaction: {enabled: true, denyKeys: [""]}`, "denyKeys"},
		{"export_extra_field", `version: "ash.obs/v0.1"
export: {allowOutbound: false, endpoint: http://evil}`, "additional"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			issues := ValidateSchema([]byte(tc.yaml))
			if len(issues) == 0 {
				t.Fatal("expected schema failure")
			}
			msg := strings.ToLower(issues[0].Message)
			if !strings.Contains(msg, strings.ToLower(tc.want)) {
				t.Fatalf("issues=%+v want substring %q", issues, tc.want)
			}
		})
	}
}

func TestValidate_semanticOutboundRequiresRedaction(t *testing.T) {
	raw := []byte(`version: "ash.obs/v0.1"
export: {allowOutbound: true}
redaction: {enabled: false}`)
	res := ParseAndValidate(raw)
	if res.OK {
		t.Fatal("expected semantic failure")
	}
	found := false
	for _, issue := range res.Issues {
		if issue.Code == "REDACTION_REQUIRED" {
			found = true
		}
	}
	if !found {
		t.Fatalf("issues=%+v", res.Issues)
	}
}

func TestValidate_otelEnabledRequiresEndpoint(t *testing.T) {
	raw := []byte(`version: "ash.obs/v0.1"
plugins:
  otel: {enabled: true}`)
	res := ParseAndValidate(raw)
	if res.OK {
		t.Fatal("expected semantic failure")
	}
}

func TestLoad_fromEnvPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "obs.yaml")
	if err := os.WriteFile(path, []byte(`version: "ash.obs/v0.1"
export: {allowOutbound: false}
redaction: {enabled: true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ASH_OBSERVABILITY_CONFIG", path)
	doc, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if doc.Version != Version {
		t.Fatalf("version=%q", doc.Version)
	}
}

func yamlMarshal(doc Document) ([]byte, error) {
	return yaml.Marshal(doc)
}
