package hooks

import "testing"

func TestEvaluate_DenyBash(t *testing.T) {
	cfg := Config{
		Version: SchemaVersion,
		Rules: []Rule{
			{Event: EventPreToolUse, Tool: "bash", Action: ActionDeny, Reason: "no shell"},
		},
	}
	dec := Evaluate(cfg, EventPreToolUse, ToolContext{Tool: "bash", Risk: "high"})
	if dec.Action != ActionDeny {
		t.Fatalf("action=%q want deny", dec.Action)
	}
	if dec.Reason != "no shell" {
		t.Fatalf("reason=%q want no shell", dec.Reason)
	}
	if dec.RuleIndex != 0 {
		t.Fatalf("ruleIndex=%d want 0", dec.RuleIndex)
	}
}

func TestEvaluate_MatchingRuleInvalidActionDenies(t *testing.T) {
	cfg := Config{
		Version: SchemaVersion,
		Rules: []Rule{
			{Event: EventPreToolUse, Tool: "bash", Action: DecisionAction("block"), Reason: "policy"},
		},
	}
	dec := Evaluate(cfg, EventPreToolUse, ToolContext{Tool: "bash", Risk: "low"})
	if dec.Action != ActionDeny {
		t.Fatalf("action=%q want deny", dec.Action)
	}
	if dec.RuleIndex != 0 {
		t.Fatalf("ruleIndex=%d want 0", dec.RuleIndex)
	}
	if dec.Reason == "" {
		t.Fatal("expected non-empty deny reason for invalid action")
	}
}

func TestEvaluate_MatchingRuleEmptyActionDenies(t *testing.T) {
	cfg := Config{
		Version: SchemaVersion,
		Rules: []Rule{
			{Event: EventPreToolUse, Tool: "*", Action: DecisionAction(""), Reason: "x"},
		},
	}
	dec := Evaluate(cfg, EventPreToolUse, ToolContext{Tool: "read_file", Risk: "low"})
	if dec.Action != ActionDeny {
		t.Fatalf("action=%q want deny", dec.Action)
	}
	if dec.RuleIndex != 0 {
		t.Fatalf("ruleIndex=%d want 0", dec.RuleIndex)
	}
}

func TestEvaluate_NoConfigAllow(t *testing.T) {
	dec := Evaluate(Config{}, EventPreToolUse, ToolContext{Tool: "read_file", Risk: "low"})
	if dec.Action != ActionAllow {
		t.Fatalf("action=%q want allow", dec.Action)
	}
	if dec.RuleIndex != -1 {
		t.Fatalf("ruleIndex=%d want -1", dec.RuleIndex)
	}
}

func TestEvaluate_postToolUseDeny(t *testing.T) {
	cfg := Config{
		Version: SchemaVersion,
		Rules: []Rule{
			{Event: EventPostToolUse, Tool: "bash", Action: ActionDeny, Reason: "cite"},
		},
	}
	dec := Evaluate(cfg, EventPostToolUse, ToolContext{Tool: "bash", Risk: "high"})
	if dec.Action != ActionDeny || dec.Reason != "cite" || dec.RuleIndex != 0 {
		t.Fatalf("%+v", dec)
	}
	pre := Evaluate(cfg, EventPreToolUse, ToolContext{Tool: "bash", Risk: "high"})
	if pre.Action != ActionAllow || pre.RuleIndex != -1 {
		t.Fatalf("pre=%+v", pre)
	}
}
