package runs

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"

	"github.com/ash-repwiki/ash/internal/agentexec"
	"github.com/ash-repwiki/ash/internal/events"
	"github.com/ash-repwiki/ash/internal/plugins"
	"github.com/ash-repwiki/ash/internal/rules"
	"github.com/ash-repwiki/ash/internal/store"
	"github.com/ash-repwiki/ash/internal/toolbus"
)

func TestExecuteAgentStepBindGateRejected(t *testing.T) {
	plugins.SetGlobalForTest(plugins.NewEmptyHost())
	t.Cleanup(plugins.ResetGlobalForTest)

	dir := t.TempDir()
	db, err := store.Open(filepath.Join(dir, "ash.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	ev := events.NewService(db)
	svc := NewService(db, ev, rules.NewLoader("scenarios"), toolbus.DefaultBus()).
		WithAgentExecutor(agentexec.StaticExecutor{})

	step := rules.Step{ID: "code.implement", Role: "Coder", Kind: "agent", TimeoutMs: 1000}
	_, err = svc.executeAgentStep("run_bind_gate", "trace_bind_gate", dir, ".", "issue", step, nil, nil, agentexec.StaticExecutor{}, "")
	if err == nil {
		t.Fatal("expected BindProduction failure on empty Global")
	}
	if !errors.Is(err, plugins.ErrUnknown) {
		t.Fatalf("want ErrUnknown, got %v", err)
	}
	if code := agentErrorCode(err); code != "AGENT_COMPONENT_UNKNOWN" {
		t.Fatalf("code=%s", code)
	}

	evs, err := svc.events.ListAfter("run_bind_gate", 0, 50)
	if err != nil {
		t.Fatal(err)
	}
	var saw bool
	for _, e := range evs {
		if e.Type != "agent.gate_rejected" {
			continue
		}
		saw = true
		var payload map[string]any
		if err := json.Unmarshal(e.Payload, &payload); err != nil {
			t.Fatal(err)
		}
		if payload["errorCode"] != "AGENT_COMPONENT_UNKNOWN" {
			t.Fatalf("payload=%v", payload)
		}
		if payload["stepId"] != "code.implement" {
			t.Fatalf("stepId=%v", payload["stepId"])
		}
	}
	if !saw {
		t.Fatal("expected agent.gate_rejected event")
	}
}
