package otel

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/ash-repwiki/ash/internal/observability"
)

// ExportWaterfall materializes waterfall spans into the active OTel tracer (batch replay).
func ExportWaterfall(ctx context.Context, wf *observability.Waterfall) (int, error) {
	if wf == nil {
		return 0, fmt.Errorf("waterfall is nil")
	}
	if len(wf.Spans) == 0 {
		return 0, nil
	}
	tracer := Tracer("ash/waterfall-export")
	ordered := orderWaterfallSpans(wf.Spans)
	contexts := map[string]trace.SpanContext{}
	exported := 0

	for _, sp := range ordered {
		start := time.UnixMilli(sp.StartTs)
		if sp.StartTs == 0 {
			start = time.Now().UTC()
		}
		end := time.UnixMilli(sp.EndTs)
		if sp.EndTs == 0 {
			end = start.Add(time.Duration(sp.DurationMs) * time.Millisecond)
		}

		parentCtx := ctx
		if sp.ParentID != "" {
			if psc, ok := contexts[sp.ParentID]; ok {
				parentCtx = trace.ContextWithSpanContext(ctx, psc)
			}
		}

		attrs := waterfallAttributes(wf, sp)
		_, span := tracer.Start(parentCtx, waterfallSpanName(sp),
			trace.WithTimestamp(start),
			trace.WithAttributes(attrs...),
			trace.WithSpanKind(trace.SpanKindInternal),
		)
		if isErrorStatus(sp.Status) {
			span.SetStatus(codes.Error, sp.Status)
		}
		span.End(trace.WithTimestamp(end))
		contexts[sp.ID] = span.SpanContext()
		exported++
	}
	return exported, nil
}

func orderWaterfallSpans(spans []observability.Span) []observability.Span {
	byID := make(map[string]observability.Span, len(spans))
	indeg := make(map[string]int, len(spans))
	children := make(map[string][]string)
	ids := make([]string, 0, len(spans))

	for _, sp := range spans {
		byID[sp.ID] = sp
		ids = append(ids, sp.ID)
	}
	sort.Strings(ids)

	for _, id := range ids {
		sp := byID[id]
		if sp.ParentID == "" || byID[sp.ParentID].ID == "" {
			continue
		}
		indeg[id]++
		children[sp.ParentID] = append(children[sp.ParentID], id)
	}

	queue := make([]string, 0, len(ids))
	for _, id := range ids {
		if indeg[id] == 0 {
			queue = append(queue, id)
		}
	}
	sort.Strings(queue)

	out := make([]observability.Span, 0, len(spans))
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		out = append(out, byID[id])
		for _, child := range children[id] {
			indeg[child]--
			if indeg[child] == 0 {
				queue = append(queue, child)
			}
		}
		sort.Strings(queue)
	}
	if len(out) != len(spans) {
		cp := append([]observability.Span(nil), spans...)
		sort.Slice(cp, func(i, j int) bool { return cp[i].StartTs < cp[j].StartTs })
		return cp
	}
	return out
}

func waterfallSpanName(sp observability.Span) string {
	switch sp.Type {
	case "run":
		return "run"
	case "step":
		if sp.Name != "" {
			return "step:" + sp.Name
		}
		return "step"
	case "tool":
		return "tool.call"
	case "model":
		return "model.chat"
	case "agent":
		return "agent.task"
	default:
		if sp.Name != "" {
			return sp.Type + ":" + sp.Name
		}
		return sp.Type
	}
}

func waterfallAttributes(wf *observability.Waterfall, sp observability.Span) []attribute.KeyValue {
	attrs := []attribute.KeyValue{
		attribute.String(AttrRunID, wf.RunID),
		attribute.String(AttrTraceID, wf.TraceID),
		attribute.String("ash.spanType", sp.Type),
		attribute.String("ash.spanStatus", sp.Status),
	}
	if sp.Name != "" {
		attrs = append(attrs, attribute.String("ash.spanName", sp.Name))
	}
	for key, val := range sp.Attributes {
		switch t := val.(type) {
		case string:
			if t != "" {
				attrs = append(attrs, attribute.String("ash."+key, t))
			}
		case float64:
			attrs = append(attrs, attribute.Float64("ash."+key, t))
		case int:
			attrs = append(attrs, attribute.Int("ash."+key, t))
		case bool:
			attrs = append(attrs, attribute.Bool("ash."+key, t))
		}
	}
	return attrs
}

func isErrorStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "failed", "error", "waiting_approval":
		return true
	default:
		return false
	}
}
