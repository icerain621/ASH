package observability

import (
	"testing"
	"time"

	"github.com/ash-repwiki/ash/internal/store"
)

func TestBuildWaterfallAggregatesSpansFailuresAndMetrics(t *testing.T) {
	db, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	done := now.Add(2 * time.Second)
	run := store.RunRecord{
		ID: "run_waterfall", TraceID: "trace_waterfall",
		ScenarioName: "feature_delivery", ScenarioVersion: "1.0.0",
		PolicyProfile: "default", Status: "failed", SpaceID: "local",
		ErrorCode: "TOOL_FAILED", ErrorMessage: "test failed",
		StartedAt: now, FinishedAt: &done, CreatedAt: now, UpdatedAt: done,
	}
	if err := db.Create(&run).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&store.RunStep{
		ID: "step_row", RunID: run.ID, StepID: "qa.verify", StepOrder: 1,
		Role: "QA", Kind: "tool_chain", Status: "failed",
		StartedAt: &now, FinishedAt: &done, DurationMs: 2000,
		ErrorCode: "TOOL_FAILED", ErrorMessage: "go test failed",
		CreatedAt: now, UpdatedAt: done,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&store.ToolCall{
		ID: "tool_row", RunID: run.ID, TraceID: run.TraceID, StepID: "qa.verify",
		Tool: "test.run", Risk: "medium", Status: "failed", Error: "exit 1",
		CreatedAt: now, CompletedAt: &done, DurationMs: 2000,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&store.AgentTask{
		ID: "agent_row", RunID: run.ID, TraceID: run.TraceID, StepID: "code.implement",
		Adapter: "codex", AgentID: "ash-codex", Status: "success",
		CreatedAt: now, StartedAt: &now, CompletedAt: &done, DurationMs: 2000,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&store.ModelUsage{
		ID: "model_row", RunID: run.ID, StepID: "arch.design",
		Provider: "fallback", Model: "reasoner", Status: "routed",
		InputTokens: 100, OutputTokens: 20, CostMicros: 42, CreatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&store.QualityMetric{
		ID: "qm_row", RunID: run.ID, SpaceID: "local",
		Name: "tool_failure_rate", Value: 1, Unit: "ratio", CreatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	for _, ev := range []store.RunEvent{
		{
			ID: "ev_artifact_quality", RunID: run.ID, Seq: 1, TS: now.UnixMilli(),
			Type: "artifact.quality_failed", Severity: "error",
			PayloadJSON: `{"error":"diff.patch is placeholder"}`, CreatedAt: now,
		},
		{
			ID: "ev_citation_missing", RunID: run.ID, Seq: 2, TS: now.UnixMilli(),
			Type: "citation.missing", Severity: "warn",
			PayloadJSON: `{"stepId":"arch.design","reason":"no repo evidence matched"}`, CreatedAt: now,
		},
		{
			ID: "ev_policy_denied", RunID: run.ID, Seq: 3, TS: now.UnixMilli(),
			Type: "policy.denied", Severity: "warn",
			PayloadJSON: `{"target":"tool","ref":"runtime.command","reason":"approval required"}`, CreatedAt: now,
		},
	} {
		if err := db.Create(&ev).Error; err != nil {
			t.Fatal(err)
		}
	}

	waterfall, err := BuildWaterfall(db, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if waterfall.RunID != run.ID || waterfall.Status != "failed" {
		t.Fatalf("waterfall=%+v want run/status", waterfall)
	}
	wantSpanTypes := map[string]bool{"run": false, "step": false, "tool": false, "agent": false, "model": false}
	for _, span := range waterfall.Spans {
		if _, ok := wantSpanTypes[span.Type]; ok {
			wantSpanTypes[span.Type] = true
		}
	}
	for typ, found := range wantSpanTypes {
		if !found {
			t.Fatalf("missing span type %q in %+v", typ, waterfall.Spans)
		}
	}
	if len(waterfall.Failures) < 3 {
		t.Fatalf("failures=%+v want run/step/tool attribution", waterfall.Failures)
	}
	wantFailures := map[string]bool{
		"artifact:artifact.quality_failed": false,
		"citation:citation.missing":        false,
		"policy:policy.denied":             false,
	}
	for _, failure := range waterfall.Failures {
		key := failure.Type + ":" + failure.Code
		if _, ok := wantFailures[key]; ok {
			wantFailures[key] = true
		}
	}
	for key, found := range wantFailures {
		if !found {
			t.Fatalf("failures=%+v missing event failure %s", waterfall.Failures, key)
		}
	}
	if len(waterfall.Metrics) != 1 || waterfall.Metrics[0].Name != "tool_failure_rate" {
		t.Fatalf("metrics=%+v want tool_failure_rate", waterfall.Metrics)
	}
}
