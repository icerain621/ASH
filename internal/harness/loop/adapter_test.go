package loop_test

import (
	"testing"

	"github.com/ash-repwiki/ash/internal/harness/loop"
	"github.com/ash-repwiki/ash/internal/sandbox"
)

type memEmitter struct {
	events []loop.EmittedEvent
}

func (m *memEmitter) Emit(runID, traceID, eventType, severity string, payload map[string]any) error {
	m.events = append(m.events, loop.EmittedEvent{
		RunID: runID, TraceID: traceID, Type: eventType, Severity: severity, Payload: payload,
	})
	return nil
}

func TestAdapterBeforeAfterToolEmitsHarnessEvents(t *testing.T) {
	em := &memEmitter{}
	ad := loop.NewAdapter(em, sandbox.NoopRouter{}, nil)
	ctx := loop.ToolHookContext{
		RunID: "run_1", TraceID: "tr_1", StepID: "s1",
		Tool: "git.status", Risk: "safe",
		ProfileDefaultMode: "workspace-write",
	}
	dec, err := ad.OnBeforeTool(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if dec.Executor != "in-process" || dec.Mode != "workspace-write" {
		t.Fatalf("decision=%+v", dec)
	}
	if err := ad.OnAfterTool(ctx, dec, true, ""); err != nil {
		t.Fatal(err)
	}
	if len(em.events) != 2 {
		t.Fatalf("events=%d want 2 %+v", len(em.events), em.events)
	}
	if em.events[0].Type != "harness.tool.routed" {
		t.Fatalf("first=%s", em.events[0].Type)
	}
	if em.events[1].Type != "harness.tool.completed" {
		t.Fatalf("second=%s", em.events[1].Type)
	}
}

func TestAssertToolResultsCovered(t *testing.T) {
	ok := loop.AssertToolResultsCovered([]string{"tool.called", "tool.result"}, []string{"git.status"})
	if !ok {
		t.Fatal("expected covered by tool.called")
	}
	ok = loop.AssertToolResultsCovered([]string{"harness.tool.completed", "tool.result"}, []string{"git.status"})
	if !ok {
		t.Fatal("expected covered by harness.tool.completed")
	}
	ok = loop.AssertToolResultsCovered([]string{"tool.result"}, []string{"git.status"})
	if ok {
		t.Fatal("expected uncovered")
	}
}
