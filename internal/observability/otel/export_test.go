package otel

import (
	"context"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/ash-repwiki/ash/internal/observability"
)

func TestExportWaterfall_emitsSpanTree(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	otel.SetTracerProvider(tp)
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })

	now := time.Now().UTC().UnixMilli()
	wf := &observability.Waterfall{
		RunID: "run_otel", TraceID: "trc_otel", Status: "completed",
		Spans: []observability.Span{
			{ID: "run:run_otel", Type: "run", Name: "feature_delivery@1.0.0", Status: "completed", StartTs: now, EndTs: now + 100, DurationMs: 100},
			{ID: "step:code.implement", ParentID: "run:run_otel", Type: "step", Name: "code.implement", Status: "finished", StartTs: now + 10, EndTs: now + 50, DurationMs: 40, Attributes: map[string]any{"role": "Coder"}},
			{ID: "tool:tc1", ParentID: "step:code.implement", Type: "tool", Name: "git.status", Status: "success", StartTs: now + 20, EndTs: now + 30, DurationMs: 10},
		},
	}
	n, err := ExportWaterfall(context.Background(), wf)
	if err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Fatalf("exported=%d want 3", n)
	}
	spans := sr.Ended()
	if len(spans) != 3 {
		t.Fatalf("ended spans=%d", len(spans))
	}
	names := map[string]int{}
	for _, sp := range spans {
		names[sp.Name()]++
	}
	for _, want := range []string{"run", "step:code.implement", "tool.call"} {
		if names[want] != 1 {
			t.Fatalf("names=%v missing %q", names, want)
		}
	}
}

func TestStartRunStepTool_liveSpans(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	otel.SetTracerProvider(tp)
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })

	ctx := context.Background()
	ctx, runSpan := StartRun(ctx, RunInfo{
		RunID: "run_live", TraceID: "trc_live", ScenarioName: "hotfix", ScenarioVersion: "1.0.0",
		PolicyProfile: "default", SpaceID: "local",
	})
	ctx, stepSpan := StartStep(ctx, "code.implement", "Coder", "tool_chain")
	ctx, toolSpan := StartToolCall(ctx, "git.status", "code.implement")
	toolSpan.End()
	stepSpan.End()
	runSpan.End()

	if len(sr.Ended()) != 3 {
		t.Fatalf("ended=%d want 3", len(sr.Ended()))
	}
}
