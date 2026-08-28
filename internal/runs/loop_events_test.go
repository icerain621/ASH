package runs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ash-repwiki/ash/internal/agentexec"
	"github.com/ash-repwiki/ash/internal/events"
	"github.com/ash-repwiki/ash/internal/harness/loop"
	"github.com/ash-repwiki/ash/internal/rules"
	"github.com/ash-repwiki/ash/internal/store"
	"github.com/ash-repwiki/ash/internal/toolbus"
)

func TestHarnessLoopEmitsRoutedAndCompleted(t *testing.T) {
	dir := t.TempDir()
	db := store.OpenTest(t, dir)
	scenariosDir := filepath.Join(dir, "scenarios")
	if err := os.MkdirAll(scenariosDir, 0o755); err != nil {
		t.Fatal(err)
	}
	scenario := `version: "ash.rules/v0.1"
scenario:
  name: "harness_loop"
  scenarioVersion: "1.0.0"
  policyProfile: "default"
  roles: { Worker: { maxParallel: 1 } }
  inputs:
    required: [issueOrSpec]
  artifacts:
    required: []
  steps:
    - id: "do.tool"
      role: "Worker"
      kind: "tool_chain"
      chain:
        - tool: "ok.tool"
          timeoutMs: 5000
`
	if err := os.WriteFile(filepath.Join(scenariosDir, "harness_loop.yaml"), []byte(scenario), 0o644); err != nil {
		t.Fatal(err)
	}
	loader := rules.NewLoader(scenariosDir)
	if err := loader.LoadDir(); err != nil {
		t.Fatal(err)
	}
	reg := toolbus.NewRegistry()
	reg.Register("ok.tool", toolbus.RiskSafe, func(_ toolbus.Context, _ map[string]any) (map[string]any, error) {
		return map[string]any{"ok": true}, nil
	})
	ev := events.NewService(db)
	svc := NewService(db, ev, loader, toolbus.NewBus(reg)).WithAgentExecutor(agentexec.StaticExecutor{})

	created, err := svc.Create(CreateRequest{
		Scenario: ScenarioRef{Name: "harness_loop", ScenarioVersion: "1.0.0"},
		Inputs:   map[string]any{"issueOrSpec": "harness loop smoke"},
	})
	if err != nil {
		t.Fatal(err)
	}

	evs, err := svc.events.ListAfter(created.RunID, 0, 200)
	if err != nil {
		t.Fatal(err)
	}
	types := make([]string, 0, len(evs))
	var sawRouted, sawCompleted, sawTurn bool
	for _, e := range evs {
		types = append(types, e.Type)
		switch e.Type {
		case "harness.tool.routed":
			sawRouted = true
		case "harness.tool.completed":
			sawCompleted = true
		case "harness.turn.started":
			sawTurn = true
		}
	}
	if !sawTurn || !sawRouted || !sawCompleted {
		t.Fatalf("missing harness events turn=%v routed=%v completed=%v types=%v", sawTurn, sawRouted, sawCompleted, types)
	}
	if !loop.AssertToolResultsCovered(types, []string{"ok.tool"}) {
		t.Fatal("HAR-02 invariant failed")
	}
}
