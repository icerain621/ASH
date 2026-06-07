package derive

import (
	"strings"
	"testing"
)

func TestReplay_runLifecycle(t *testing.T) {
	events := []Event{
		{
			RunID: "run_1",
			Type:  "run.started",
			PayloadJSON: `{
				"scenario":{"name":"feature_delivery","scenarioVersion":"1.0.0"},
				"policyProfile":"default","inputsDigest":"d1"
			}`,
		},
		{
			RunID: "run_1",
			Type:  "step.finished",
			PayloadJSON: `{"stepId":"code.implement","role":"Coder","ok":true,"durationMs":1200}`,
		},
		{
			RunID: "run_1",
			Type:  "run.finished",
			PayloadJSON: `{"ok":true,"durationMs":5000,"artifacts":[{"id":"a1","type":"diff","digest":"sha256:x"}]}`,
		},
	}
	snap := Replay(events)
	if snap.Counters[`ash_run_total{scenario="feature_delivery",status="started"}`] != 1 {
		t.Fatalf("started counter=%v", snap.Counters)
	}
	if snap.Counters[`ash_run_total{scenario="feature_delivery",status="finished"}`] != 1 {
		t.Fatalf("finished counter=%v", snap.Counters)
	}
	if snap.Gauges[`ash_run_inflight{scenario="feature_delivery"}`] != 0 {
		t.Fatalf("inflight=%v want 0", snap.Gauges)
	}
	if len(snap.Histograms[`ash_run_duration_ms{scenario="feature_delivery"}`]) != 1 {
		t.Fatalf("duration hist=%v", snap.Histograms)
	}
	if snap.Counters[`ash_step_total{scenario="feature_delivery",status="ok",stepId="code.implement"}`] != 1 {
		t.Fatalf("step counter=%v", snap.Counters)
	}
}

func TestReplay_toolAndPolicy(t *testing.T) {
	events := []Event{
		{
			RunID: "run_1", Type: "run.started",
			PayloadJSON: `{"scenario":{"name":"hotfix","scenarioVersion":"1"},"policyProfile":"strict","inputsDigest":"d"}`,
		},
		{
			RunID: "run_1", Type: "tool.result",
			PayloadJSON: `{"tool":"git.status","ok":false,"durationMs":42}`,
		},
		{
			RunID: "run_1", Type: "policy.denied",
			PayloadJSON: `{"target":"tool","reason":"POLICY_DENIED","action":"deny"}`,
		},
	}
	snap := Replay(events)
	if snap.Counters[`ash_tool_calls_total{ok="false",tool="git.status"}`] != 1 {
		t.Fatalf("tool counter=%v", snap.Counters)
	}
	if snap.Counters[`ash_policy_denied_total{reason="POLICY_DENIED",target="tool"}`] != 1 {
		t.Fatalf("policy counter=%v", snap.Counters)
	}
}

func TestReplay_memoryAndModel(t *testing.T) {
	events := []Event{
		{RunID: "r", Type: "memory.candidate_created", PayloadJSON: `{"candidateId":"c1","layer":"L1","evidenceCount":2,"sensitivity":"normal"}`},
		{RunID: "r", Type: "memory.reviewed", PayloadJSON: `{"candidateId":"c1","layer":"L1","decision":"approve","reason":"ok","policyProfile":"default","latencyMs":80}`},
		{RunID: "r", Type: "model.usage", PayloadJSON: `{"providerId":"openai","modelId":"gpt","inTokens":10,"outTokens":5}`},
	}
	snap := Replay(events)
	if snap.Counters[`ash_memory_candidates_total{layer="L1"}`] != 1 {
		t.Fatal(snap.Counters)
	}
	if snap.Counters[`ash_memory_reviews_total{decision="approve",layer="L1"}`] != 1 {
		t.Fatal(snap.Counters)
	}
	if snap.Counters[`ash_token_in_total{model="gpt",provider="openai"}`] != 10 {
		t.Fatal(snap.Counters)
	}
	if snap.Counters[`ash_token_out_total{model="gpt",provider="openai"}`] != 5 {
		t.Fatal(snap.Counters)
	}
}

func TestCatalog_coversAppendixCoreEvents(t *testing.T) {
	want := []string{
		"run.started", "run.finished", "run.failed", "step.finished",
		"tool.result", "policy.denied", "model.usage", "rag.results",
		"memory.candidate_created", "memory.reviewed",
	}
	seen := map[string]struct{}{}
	for _, rule := range Catalog() {
		seen[rule.EventType] = struct{}{}
	}
	for _, ev := range want {
		if _, ok := seen[ev]; !ok {
			t.Fatalf("catalog missing event type %q", ev)
		}
	}
}

func TestSnapshot_PrometheusText(t *testing.T) {
	snap := Replay([]Event{{
		RunID: "r", Type: "run.started",
		PayloadJSON: `{"scenario":{"name":"x","scenarioVersion":"1"},"policyProfile":"default","inputsDigest":"d"}`,
	}})
	text := snap.PrometheusText()
	if !strings.Contains(text, "ash_run_total") {
		t.Fatalf("text=%q", text)
	}
}
