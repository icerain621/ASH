package hooks

import (
	"strings"
	"testing"
)

func TestParse_BadVersion(t *testing.T) {
	_, err := ParseConfigJSON(`{"version":"hooks.v0","rules":[]}`)
	if err == nil {
		t.Fatal("expected error for bad version")
	}
	if !strings.Contains(err.Error(), SchemaVersion) {
		t.Fatalf("error=%v want mention of schema version", err)
	}
}

func TestConfigFromSpaceBodyJSON_NoHooksAllow(t *testing.T) {
	cfg, err := ConfigFromSpaceBodyJSON(`{"citationMode":"optional"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Rules) != 0 {
		t.Fatalf("rules=%d want 0", len(cfg.Rules))
	}
	dec := Evaluate(cfg, EventPreToolUse, ToolContext{Tool: "bash"})
	if dec.Action != ActionAllow || dec.RuleIndex != -1 {
		t.Fatalf("decision=%+v want default allow", dec)
	}
}
