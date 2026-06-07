package rules

import (
	"strings"
	"testing"
)

func TestValidateSchema_bundledScenarios(t *testing.T) {
	t.Parallel()
	TestParseAndValidate_bundledScenarios(t)
}

func TestValidateSchema_invalidSamples(t *testing.T) {
	cases := []struct {
		name string
		yaml string
		want string
	}{
		{"wrong_version", `version: "ash.rules/v0.0"
scenario:
  name: x
  scenarioVersion: "1"
  steps: [{id: s1, role: PM, kind: llm, promptRef: p.md}]`, "version"},
		{"missing_scenario", `version: "ash.rules/v0.1"`, "scenario"},
		{"empty_steps", `version: "ash.rules/v0.1"
scenario:
  name: x
  scenarioVersion: "1"
  steps: []`, "steps"},
		{"invalid_kind", `version: "ash.rules/v0.1"
scenario:
  name: x
  scenarioVersion: "1"
  steps: [{id: s1, role: PM, kind: magic, promptRef: p.md}]`, "kind"},
		{"invalid_policy", `version: "ash.rules/v0.1"
scenario:
  name: x
  scenarioVersion: "1"
  policyProfile: loose
  steps: [{id: s1, role: PM, kind: llm, promptRef: p.md}]`, "policyProfile"},
		{"invalid_hook_event", `version: "ash.rules/v0.1"
scenario:
  name: x
  scenarioVersion: "1"
  steps: [{id: s1, role: PM, kind: llm, promptRef: p.md}]
hooks:
  - {id: h1, on: run.hacked, policy: enforce, rules: [{match: {tool: git.status}, action: {deny: true}}]}`, "on"},
		{"rag_missing_requireCitations", `version: "ash.rules/v0.1"
scenario:
  name: x
  scenarioVersion: "1"
  steps:
    - id: s1
      role: PM
      kind: llm
      promptRef: p.md
      rag:
        sources: [code]`, "requireCitations"},
		{"unknown_top_level", `version: "ash.rules/v0.1"
extra: true
scenario:
  name: x
  scenarioVersion: "1"
  steps: [{id: s1, role: PM, kind: llm, promptRef: p.md}]`, "additional"},
		{"gate_missing_when", `version: "ash.rules/v0.1"
scenario:
  name: x
  scenarioVersion: "1"
  gates: [{id: g1, blocking: true, check: {tool: git.status}}]
  steps: [{id: s1, role: PM, kind: llm, promptRef: p.md}]`, "when"},
		{"invalid_checkpoint", `version: "ash.rules/v0.1"
scenario:
  name: x
  scenarioVersion: "1"
  checkpoint: {strategy: never, retain: -1}
  steps: [{id: s1, role: PM, kind: llm, promptRef: p.md}]`, "strategy"},
		{"hook_missing_rules", `version: "ash.rules/v0.1"
scenario:
  name: x
  scenarioVersion: "1"
  steps: [{id: s1, role: PM, kind: llm, promptRef: p.md}]
hooks:
  - {id: h1, on: tool.called, policy: enforce, rules: []}`, "rules"},
		{"step_missing_role", `version: "ash.rules/v0.1"
scenario:
  name: x
  scenarioVersion: "1"
  steps: [{id: s1, kind: llm, promptRef: p.md}]`, "role"},
		{"invalid_missing_citations_action", `version: "ash.rules/v0.1"
scenario:
  name: x
  scenarioVersion: "1"
  steps:
    - id: s1
      role: PM
      kind: llm
      promptRef: p.md
      rag:
        sources: [code]
        requireCitations: true
        onMissingCitations: ignore`, "onMissingCitations"},
		{"agent_step_extra_field", `version: "ash.rules/v0.1"
scenario:
  name: x
  scenarioVersion: "1"
  steps:
    - id: s1
      role: PM
      kind: agent
      agent: {adapter: codex, secret: leak}`, "additional"},
		{"tool_chain_missing_tool", `version: "ash.rules/v0.1"
scenario:
  name: x
  scenarioVersion: "1"
  steps:
    - id: s1
      role: PM
      kind: tool_chain
      chain: [{args: {}}]`, "tool"},
		{"memory_candidate_missing_layer", `version: "ash.rules/v0.1"
scenario:
  name: x
  scenarioVersion: "1"
  steps:
    - id: s1
      role: PM
      kind: llm
      promptRef: p.md
      outputs:
        memoryCandidates: [{title: x}]`, "layer"},
		{"invalid_hook_policy", `version: "ash.rules/v0.1"
scenario:
  name: x
  scenarioVersion: "1"
  steps: [{id: s1, role: PM, kind: llm, promptRef: p.md}]
hooks:
  - {id: h1, on: tool.called, policy: warn, rules: [{match: {tool: git.status}, action: {deny: true}}]}`, "policy"},
		{"gate_on_fail_missing_message", `version: "ash.rules/v0.1"
scenario:
  name: x
  scenarioVersion: "1"
  gates:
    - id: g1
      when: run.before_finish
      blocking: true
      check: {tool: git.status}
      onFail: {}
  steps: [{id: s1, role: PM, kind: llm, promptRef: p.md}]`, "message"},
		{"scenario_extra_field", `version: "ash.rules/v0.1"
scenario:
  name: x
  scenarioVersion: "1"
  owner: someone
  steps: [{id: s1, role: PM, kind: llm, promptRef: p.md}]`, "additional"},
		{"invalid_retry", `version: "ash.rules/v0.1"
scenario:
  name: x
  scenarioVersion: "1"
  steps:
    - id: s1
      role: PM
      kind: llm
      promptRef: p.md
      retry: {maxAttempts: 0}`, "maxAttempts"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			issues := ValidateSchema([]byte(tc.yaml))
			if len(issues) == 0 {
				t.Fatalf("expected schema failure")
			}
			msg := strings.ToLower(issues[0].Message)
			if !strings.Contains(msg, strings.ToLower(tc.want)) {
				t.Fatalf("issues=%+v want substring %q", issues, tc.want)
			}
		})
	}
}
