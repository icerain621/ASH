package runs

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ash-repwiki/ash/internal/agentexec"
	"github.com/ash-repwiki/ash/internal/events"
	"github.com/ash-repwiki/ash/internal/harness"
	"github.com/ash-repwiki/ash/internal/rules"
	"github.com/ash-repwiki/ash/internal/sandbox"
	"github.com/ash-repwiki/ash/internal/store"
	"github.com/ash-repwiki/ash/internal/toolbus"
)

func TestToolOutputSpillAndCompaction(t *testing.T) {
	dir := t.TempDir()
	db := store.OpenTest(t, dir)
	scenariosDir := filepath.Join(dir, "scenarios")
	if err := os.MkdirAll(scenariosDir, 0o755); err != nil {
		t.Fatal(err)
	}
	scenario := `version: "ash.rules/v0.1"
scenario:
  name: "spill_compact"
  scenarioVersion: "1.0.0"
  policyProfile: "default"
  roles: { Worker: { maxParallel: 1 } }
  inputs:
    required: [issueOrSpec]
  artifacts:
    required: []
  steps:
    - id: "do.big"
      role: "Worker"
      kind: "tool_chain"
      chain:
        - tool: "big.tool"
          timeoutMs: 5000
`
	if err := os.WriteFile(filepath.Join(scenariosDir, "spill_compact.yaml"), []byte(scenario), 0o644); err != nil {
		t.Fatal(err)
	}
	loader := rules.NewLoader(scenariosDir)
	if err := loader.LoadDir(); err != nil {
		t.Fatal(err)
	}
	big := strings.Repeat("x", 14000)
	reg := toolbus.NewRegistry()
	reg.Register("big.tool", toolbus.RiskSafe, func(_ toolbus.Context, _ map[string]any) (map[string]any, error) {
		return map[string]any{"blob": big}, nil
	})
	ev := events.NewService(db)
	svc := NewService(db, ev, loader, toolbus.NewBus(reg)).
		WithAgentExecutor(agentexec.StaticExecutor{}).
		WithSandboxRouter(sandbox.NoopRouter{})

	spec := harness.DefaultSpec()
	spec.Sandbox.SpillMaxBytes = 1024
	spec.Compaction = &harness.CompactionSpec{Enabled: true, TriggerTokenRatio: 0.1}
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
		Scenario: ScenarioRef{Name: "spill_compact", ScenarioVersion: "1.0.0"},
		Inputs:   map[string]any{"issueOrSpec": "spill"},
	})
	if err != nil {
		t.Fatal(err)
	}
	evs, err := svc.events.ListAfter(created.RunID, 0, 300)
	if err != nil {
		t.Fatal(err)
	}
	sawSpill, sawCompact := false, false
	for _, e := range evs {
		switch e.Type {
		case "tool.spilled":
			sawSpill = true
		case "harness.compaction":
			sawCompact = true
		case "tool.result":
			var payload map[string]any
			_ = json.Unmarshal(e.Payload, &payload)
			out, _ := payload["output"].(map[string]any)
			if out != nil && out["spilled"] != true {
				t.Fatalf("tool.result output should be spilled summary: %v", out)
			}
		}
	}
	if !sawSpill {
		t.Fatal("missing tool.spilled")
	}
	if !sawCompact {
		t.Fatal("missing harness.compaction")
	}
	art := filepath.Join(db.RunDir(created.RunID), "artifacts")
	entries, _ := os.ReadDir(art)
	found := false
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "spill_") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected spill artifact in %s", art)
	}
}
