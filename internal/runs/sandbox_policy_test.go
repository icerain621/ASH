package runs

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ash-repwiki/ash/internal/agentexec"
	"github.com/ash-repwiki/ash/internal/events"
	"github.com/ash-repwiki/ash/internal/harness"
	"github.com/ash-repwiki/ash/internal/rules"
	"github.com/ash-repwiki/ash/internal/sandbox"
	"github.com/ash-repwiki/ash/internal/store"
	"github.com/ash-repwiki/ash/internal/toolbus"
)

func TestSandboxRaisesDangerToIsolatedWhenProfileOff(t *testing.T) {
	dir := t.TempDir()
	db := store.OpenTest(t, dir)
	scenariosDir := filepath.Join(dir, "scenarios")
	if err := os.MkdirAll(scenariosDir, 0o755); err != nil {
		t.Fatal(err)
	}
	scenario := `version: "ash.rules/v0.1"
scenario:
  name: "sandbox_raise"
  scenarioVersion: "1.0.0"
  policyProfile: "default"
  roles: { Worker: { maxParallel: 1 } }
  inputs:
    required: [issueOrSpec]
  artifacts:
    required: []
  steps:
    - id: "do.danger"
      role: "Worker"
      kind: "tool_chain"
      chain:
        - tool: "danger.tool"
          timeoutMs: 5000
          policy: "allow_dangerous"
`
	if err := os.WriteFile(filepath.Join(scenariosDir, "sandbox_raise.yaml"), []byte(scenario), 0o644); err != nil {
		t.Fatal(err)
	}
	loader := rules.NewLoader(scenariosDir)
	if err := loader.LoadDir(); err != nil {
		t.Fatal(err)
	}
	reg := toolbus.NewRegistry()
	reg.Register("danger.tool", toolbus.RiskDanger, func(_ toolbus.Context, _ map[string]any) (map[string]any, error) {
		return map[string]any{"ok": true}, nil
	})
	ev := events.NewService(db)
	svc := NewService(db, ev, loader, toolbus.NewBus(reg)).
		WithAgentExecutor(agentexec.StaticExecutor{}).
		WithSandboxRouter(sandbox.NoopRouter{})

	spec := harness.DefaultSpec()
	spec.Sandbox.DefaultMode = sandbox.ModeOff
	createdProf, err := svc.harnessSvc.Create(harness.CreateRequest{Name: "default", Spec: spec, CreatedBy: "tester"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.harnessSvc.SubmitReview(createdProf.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.harnessSvc.Promote(createdProf.ID, "tester"); err != nil {
		t.Fatal(err)
	}

	created, err := svc.Create(CreateRequest{
		Scenario: ScenarioRef{Name: "sandbox_raise", ScenarioVersion: "1.0.0"},
		Inputs:   map[string]any{"issueOrSpec": "raise danger to isolated"},
	})
	if err != nil {
		t.Fatal(err)
	}
	evs, listErr := svc.events.ListAfter(created.RunID, 0, 200)
	if listErr != nil {
		t.Fatal(listErr)
	}
	sawIsolated := false
	for _, e := range evs {
		if e.Type != "harness.tool.routed" {
			continue
		}
		var payload map[string]any
		_ = json.Unmarshal(e.Payload, &payload)
		if payload["sandboxMode"] == sandbox.ModeIsolated && payload["risk"] == "danger" {
			sawIsolated = true
			break
		}
	}
	if !sawIsolated {
		t.Fatalf("expected harness.tool.routed with isolated for danger")
	}
}

func TestHotfixPolicyForcesIsolatedForSafeTool(t *testing.T) {
	dir := t.TempDir()
	db := store.OpenTest(t, dir)
	scenariosDir := filepath.Join(dir, "scenarios")
	if err := os.MkdirAll(scenariosDir, 0o755); err != nil {
		t.Fatal(err)
	}
	scenario := `version: "ash.rules/v0.1"
scenario:
  name: "hotfix_force"
  scenarioVersion: "1.0.0"
  policyProfile: "hotfix"
  sandbox:
    minMode: isolated
  roles: { Worker: { maxParallel: 1 } }
  inputs:
    required: [issueOrSpec]
  artifacts:
    required: []
  steps:
    - id: "do.read"
      role: "Worker"
      kind: "tool_chain"
      chain:
        - tool: "safe.tool"
          timeoutMs: 5000
`
	if err := os.WriteFile(filepath.Join(scenariosDir, "hotfix_force.yaml"), []byte(scenario), 0o644); err != nil {
		t.Fatal(err)
	}
	loader := rules.NewLoader(scenariosDir)
	if err := loader.LoadDir(); err != nil {
		t.Fatal(err)
	}
	reg := toolbus.NewRegistry()
	reg.Register("safe.tool", toolbus.RiskSafe, func(_ toolbus.Context, _ map[string]any) (map[string]any, error) {
		return map[string]any{"ok": true}, nil
	})
	ev := events.NewService(db)
	svc := NewService(db, ev, loader, toolbus.NewBus(reg)).
		WithAgentExecutor(agentexec.StaticExecutor{}).
		WithSandboxRouter(sandbox.NoopRouter{})

	spec := harness.DefaultSpec()
	spec.Sandbox.DefaultMode = sandbox.ModeWorkspaceWrite
	createdProf, err := svc.harnessSvc.Create(harness.CreateRequest{Name: "default", Spec: spec, CreatedBy: "tester"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.harnessSvc.SubmitReview(createdProf.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.harnessSvc.Promote(createdProf.ID, "tester"); err != nil {
		t.Fatal(err)
	}

	created, err := svc.Create(CreateRequest{
		Scenario:      ScenarioRef{Name: "hotfix_force", ScenarioVersion: "1.0.0"},
		PolicyProfile: "hotfix",
		Inputs:        map[string]any{"issueOrSpec": "force isolated"},
	})
	if err != nil {
		t.Fatal(err)
	}
	evs, err := svc.events.ListAfter(created.RunID, 0, 200)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range evs {
		if e.Type != "harness.tool.routed" {
			continue
		}
		var payload map[string]any
		_ = json.Unmarshal(e.Payload, &payload)
		if payload["sandboxMode"] != sandbox.ModeIsolated {
			t.Fatalf("payload=%v want isolated", payload)
		}
		return
	}
	t.Fatal("missing harness.tool.routed")
}
