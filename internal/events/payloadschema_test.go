package events

import (
	"testing"
)

func TestValidatePayload_runStarted(t *testing.T) {
	raw := []byte(`{
		"scenario": {"name": "feature_delivery", "scenarioVersion": "1.0.0"},
		"policyProfile": "default",
		"inputsDigest": "sha256:abc"
	}`)
	if err := ValidatePayload("run.started", raw); err != nil {
		t.Fatal(err)
	}
}

func TestValidatePayload_invalidRunStarted(t *testing.T) {
	raw := []byte(`{"policyProfile": "default"}`)
	if err := ValidatePayload("run.started", raw); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestValidatePayload_unknownEventSkipped(t *testing.T) {
	raw := []byte(`{"status": "canceled"}`)
	if err := ValidatePayload("run.canceled", raw); err != nil {
		t.Fatal(err)
	}
}

func TestValidatePayload_invalidSamples(t *testing.T) {
	cases := []struct {
		name      string
		eventType string
		json      string
	}{
		{"tool_missing_risk", "tool.called", `{"tool":"git.status","timeoutMs":1,"argsDigest":"d"}`},
		{"run_failed_ok_true", "run.failed", `{"ok":true,"durationMs":1,"error":{"code":"x","message":"y"}}`},
		{"step_started_bad_kind", "step.started", `{"stepId":"s1","role":"PM","kind":"magic"}`},
		{"policy_denied_bad_target", "policy.denied", `{"target":"x","reason":"y","action":"deny"}`},
		{"memory_review_missing_reason", "memory.reviewed", `{"candidateId":"c1","decision":"approve","policyProfile":"default"}`},
		{"rag_query_empty_sources", "rag.query", `{"queryDigest":"d","sources":[],"topK":3,"requireCitations":true}`},
		{"checkpoint_missing_strategy", "run.checkpoint_saved", `{"checkpointId":"c","stepId":"s","snapshotDigest":"d"}`},
		{"model_usage_negative_tokens", "model.usage", `{"providerId":"p","modelId":"m","inTokens":-1,"outTokens":0}`},
		{"run_finished_no_artifacts", "run.finished", `{"ok":true,"durationMs":1,"artifacts":[]}`},
		{"tool_result_missing_duration", "tool.result", `{"tool":"git.status","ok":true}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidatePayload(tc.eventType, []byte(tc.json)); err == nil {
				t.Fatalf("expected validation error for %s", tc.eventType)
			}
		})
	}
}

func TestKnownTR0EventTypes(t *testing.T) {
	types, err := KnownTR0EventTypes()
	if err != nil {
		t.Fatal(err)
	}
	if len(types) < 10 {
		t.Fatalf("types=%v", types)
	}
}
