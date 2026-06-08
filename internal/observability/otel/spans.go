package otel

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// RunInfo carries root span attributes for a run execution.
type RunInfo struct {
	RunID           string
	TraceID         string
	ScenarioName    string
	ScenarioVersion string
	PolicyProfile   string
	SpaceID         string
}

// StartRun begins the root `run` span (appendix D §5).
func StartRun(ctx context.Context, info RunInfo) (context.Context, trace.Span) {
	attrs := runAttrs(info.RunID, info.TraceID, info.ScenarioName, info.ScenarioVersion, info.PolicyProfile, info.SpaceID)
	return Tracer("ash/run").Start(ctx, "run", trace.WithSpanKind(trace.SpanKindServer), trace.WithAttributes(attrs...))
}

// StartStep begins a `step:*` child span.
func StartStep(ctx context.Context, stepID, role, kind string) (context.Context, trace.Span) {
	return Tracer("ash/step").Start(ctx, "step:"+stepID,
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(
			attribute.String(AttrStepID, stepID),
			attribute.String(AttrRole, role),
			attribute.String("ash.stepKind", kind),
		),
	)
}

// StartGate begins a `gate:*` child span.
func StartGate(ctx context.Context, gateID string) (context.Context, trace.Span) {
	return Tracer("ash/gate").Start(ctx, "gate:"+gateID,
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(attribute.String(AttrGateID, gateID)),
	)
}

// StartRAGQuery begins a `rag.query` span.
func StartRAGQuery(ctx context.Context, runID, stepID string) (context.Context, trace.Span) {
	return Tracer("ash/rag").Start(ctx, "rag.query",
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(
			attribute.String(AttrRunID, runID),
			attribute.String(AttrStepID, stepID),
		),
	)
}

// StartModelChat begins a `model.chat` span.
func StartModelChat(ctx context.Context, provider, model, stepID string) (context.Context, trace.Span) {
	return Tracer("ash/model").Start(ctx, "model.chat",
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(
			attribute.String(AttrProvider, provider),
			attribute.String(AttrModel, model),
			attribute.String(AttrStepID, stepID),
		),
	)
}

// StartToolCall begins a `tool.call` span.
func StartToolCall(ctx context.Context, tool, stepID string) (context.Context, trace.Span) {
	return Tracer("ash/tool").Start(ctx, "tool.call",
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(
			attribute.String(AttrTool, tool),
			attribute.String(AttrStepID, stepID),
		),
	)
}

// EndSpan ends a span and records error status when needed.
func EndSpan(span trace.Span, err error) {
	if span == nil {
		return
	}
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	span.End()
}
