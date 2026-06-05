package improve

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ash-repwiki/ash/internal/agentexec"
	"github.com/ash-repwiki/ash/internal/events"
	"github.com/ash-repwiki/ash/internal/rules"
	"github.com/ash-repwiki/ash/internal/runs"
	"github.com/ash-repwiki/ash/internal/store"
	"github.com/ash-repwiki/ash/internal/toolbus"
)

func TestImproveProposalExperimentFlow(t *testing.T) {
	dir := t.TempDir()
	db, err := store.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	scenariosDir := filepath.Join("..", "..", "scenarios")
	if _, err := os.Stat(scenariosDir); err != nil {
		scenariosDir = "scenarios"
	}
	loader := rules.NewLoader(scenariosDir)
	if err := loader.LoadDir(); err != nil {
		t.Fatal(err)
	}
	ev := events.NewService(db)
	runsSvc := runs.NewService(db, ev, loader, toolbus.DefaultBus()).WithAgentExecutor(agentexec.StaticExecutor{})
	svc := NewService(db, runsSvc, ev)

	baseline, err := runsSvc.Create(runs.CreateRequest{
		Scenario: runs.ScenarioRef{Name: "feature_delivery", ScenarioVersion: "1.0.0"},
		Inputs: map[string]any{
			"issueOrSpec": "improve baseline",
			"repoRoot":    dir,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	proposal, err := svc.Create(CreateProposalRequest{
		Title:         "Replay compare",
		BaselineRunID: baseline.RunID,
		ChangeSummary: "doctor improve flow",
	})
	if err != nil {
		t.Fatal(err)
	}
	if proposal.Status != "draft" {
		t.Fatalf("status=%q want draft", proposal.Status)
	}

	exp, err := svc.StartExperiment("", proposal.ID)
	if err != nil {
		t.Fatal(err)
	}
	if exp.ExperimentRunID == "" || exp.Compare == nil {
		t.Fatalf("experiment=%+v", exp)
	}

	canary, err := svc.StartCanary("", proposal.ID, CanaryRequest{Percent: 10})
	if err != nil {
		t.Fatal(err)
	}
	if canary.Status != "canary" {
		t.Fatalf("canary status=%q", canary.Status)
	}

	promoted, err := svc.Promote("", proposal.ID)
	if err != nil {
		t.Fatal(err)
	}
	if promoted.Status != "promoted" {
		t.Fatalf("promoted status=%q", promoted.Status)
	}
}
