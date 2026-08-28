package runs

import (
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

func TestSandboxDeniesDangerWhenProfileOff(t *testing.T) {
	dir := t.TempDir()
	db := store.OpenTest(t, dir)
	scenariosDir := filepath.Join(dir, "scenarios")
	if err := os.MkdirAll(scenariosDir, 0o755); err != nil {
		t.Fatal(err)
	}
	scenario := `version: "ash.rules/v0.1"
scenario:
  name: "sandbox_deny"
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
	if err := os.WriteFile(filepath.Join(scenariosDir, "sandbox_deny.yaml"), []byte(scenario), 0o644); err != nil {
		t.Fatal(err)
	}
	loader := rules.NewLoader(scenariosDir)
	if err := loader.LoadDir(); err != nil {
		t.Fatal(err)
	}
	reg := toolbus.NewRegistry()
	reg.Register("danger.tool", toolbus.RiskDanger, func(_ toolbus.Context, _ map[string]any) (map[string]any, error) {
		t.Fatal("danger tool must not execute under off")
		return nil, nil
	})
	ev := events.NewService(db)
	svc := NewService(db, ev, loader, toolbus.NewBus(reg)).WithAgentExecutor(agentexec.StaticExecutor{})

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
		Scenario: ScenarioRef{Name: "sandbox_deny", ScenarioVersion: "1.0.0"},
		Inputs:   map[string]any{"issueOrSpec": "deny danger under off"},
	})
	if created == nil {
		t.Fatalf("expected run id even on tool failure, err=%v", err)
	}
	evs, listErr := svc.events.ListAfter(created.RunID, 0, 200)
	if listErr != nil {
		t.Fatal(listErr)
	}
	sawDeny := false
	for _, e := range evs {
		if e.Type == "policy.denied" {
			sawDeny = true
			break
		}
	}
	if !sawDeny {
		types := make([]string, 0, len(evs))
		for _, e := range evs {
			types = append(types, e.Type)
		}
		t.Fatalf("expected policy.denied, events=%v createErr=%v", types, err)
	}
}
