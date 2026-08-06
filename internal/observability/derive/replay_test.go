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

func TestReplay_runCanceledClearsInflight(t *testing.T) {
	events := []Event{
		{
			RunID: "run_c",
			Type:  "run.started",
			PayloadJSON: `{
				"scenario":{"name":"feature_delivery","scenarioVersion":"1.0.0"},
				"policyProfile":"default","inputsDigest":"d1"
			}`,
		},
		{
			RunID:       "run_c",
			Type:        "run.canceled",
			PayloadJSON: `{"status":"canceled"}`,
		},
	}
	snap := Replay(events)
	if snap.Counters[`ash_run_total{scenario="feature_delivery",status="canceled"}`] != 1 {
		t.Fatalf("canceled counter=%v", snap.Counters)
	}
	if snap.Gauges[`ash_run_inflight{scenario="feature_delivery"}`] != 0 {
		t.Fatalf("inflight=%v want 0 after cancel", snap.Gauges)
	}
	if err := ValidateReplayParity(events); err != nil {
		t.Fatal(err)
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
	if snap.Gauges[`ash_memory_unreviewed_backlog{layer="L1"}`] != 0 {
		t.Fatalf("backlog gauge=%v want 0 after review", snap.Gauges)
	}
	if snap.Counters[`ash_token_in_total{model="gpt",provider="openai"}`] != 10 {
		t.Fatal(snap.Counters)
	}
	if snap.Counters[`ash_token_out_total{model="gpt",provider="openai"}`] != 5 {
		t.Fatal(snap.Counters)
	}
}

func TestReplay_memoryMissingEvidenceAndBacklog(t *testing.T) {
	events := []Event{
		{RunID: "r", Type: "memory.candidate_created", PayloadJSON: `{"candidateId":"c0","layer":"L2","evidenceCount":0,"sensitivity":"normal"}`},
		{RunID: "r", Type: "memory.candidate_created", PayloadJSON: `{"candidateId":"c1","layer":"L2","evidenceCount":1,"sensitivity":"normal"}`},
	}
	snap := Replay(events)
	if snap.Counters[`ash_memory_missing_evidence_total{layer="L2"}`] != 1 {
		t.Fatalf("missing evidence=%v", snap.Counters)
	}
	if snap.Gauges[`ash_memory_unreviewed_backlog{layer="L2"}`] != 2 {
		t.Fatalf("backlog=%v want 2", snap.Gauges)
	}
}

func TestReplay_memoryTTLExpired(t *testing.T) {
	events := []Event{{
		RunID: "r", Type: "memory.ttl_expired",
		PayloadJSON: `{"memoryId":"m1","layer":"L1","reason":"ttl_expired"}`,
	}}
	snap := Replay(events)
	if snap.Counters[`ash_memory_ttl_expired_total{layer="L1",reason="ttl_expired"}`] != 1 {
		t.Fatalf("ttl expired=%v", snap.Counters)
	}
}

func TestReplay_memoryHitDeprecatedQuery(t *testing.T) {
	events := []Event{
		{RunID: "r", Type: "memory.hit_used", PayloadJSON: `{"count":3,"hitsByLayer":{"L1":2,"L2":1}}`},
		{RunID: "r", Type: "memory.deprecated", PayloadJSON: `{"memoryId":"m1","layer":"L1","reason":"stale"}`},
		{RunID: "r", Type: "memory.query", PayloadJSON: `{"layersKey":"L1,L2","resultCount":2,"latencyMs":45}`},
	}
	snap := Replay(events)
	if snap.Counters[`ash_memory_hit_used_total{layer="L1"}`] != 2 {
		t.Fatalf("L1 hits=%v", snap.Counters)
	}
	if snap.Counters[`ash_memory_hit_used_total{layer="L2"}`] != 1 {
		t.Fatalf("L2 hits=%v", snap.Counters)
	}
	if snap.Counters[`ash_memory_deprecated_total{layer="L1",reason="stale"}`] != 1 {
		t.Fatalf("deprecated=%v", snap.Counters)
	}
	if snap.Counters[`ash_memory_queries_total{layersKey="L1,L2"}`] != 1 {
		t.Fatalf("queries=%v", snap.Counters)
	}
	if len(snap.Histograms[`ash_memory_query_latency_ms{layersKey="L1,L2"}`]) != 1 {
		t.Fatalf("query latency=%v", snap.Histograms)
	}
}

func TestReplay_ragRetrievedFallback(t *testing.T) {
	snap := Replay([]Event{{
		RunID: "r", Type: "rag.retrieved",
		PayloadJSON: `{"retrievalMode":"chunk","hits":2}`,
	}})
	if snap.Counters[`ash_rag_queries_total{mode="chunk"}`] != 1 {
		t.Fatalf("queries counter=%v", snap.Counters)
	}
	if snap.Counters[`ash_rag_fts_fallback_total{mode="chunk"}`] != 1 {
		t.Fatalf("fallback counter=%v", snap.Counters)
	}
}

func TestReplay_memoryMigrated(t *testing.T) {
	snap := Replay([]Event{{
		RunID: "r", Type: "memory.migrated",
		PayloadJSON: `{"from":0,"to":1,"ok":true,"recordsUpdated":3,"summary":"v0→v1"}`,
	}})
	key := `ash_memory_migration_runs_total{from="0",ok="true",to="1"}`
	if snap.Counters[key] != 1 {
		t.Fatalf("counter=%v want key %s", snap.Counters, key)
	}
}

func TestCatalog_coversAppendixCoreEvents(t *testing.T) {
	want := []string{
		"run.started", "run.finished", "run.failed", "run.canceled", "step.finished",
		"tool.result", "policy.denied", "model.usage", "rag.results", "rag.retrieved",
		"memory.candidate_created", "memory.reviewed",
		"memory.hit_used", "memory.deprecated", "memory.query", "memory.migrated",
		"memory.ttl_expired", "memory.confidence_adjusted",
		"plugin.export_failed",
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
