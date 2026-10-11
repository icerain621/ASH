package agenttpl

import (
	"errors"
	"testing"

	"github.com/ash-repwiki/ash/internal/plugins"
)

func TestBuiltinTemplatesLoad(t *testing.T) {
	list, err := List()
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Items) != 5 {
		t.Fatalf("templates=%d", len(list.Items))
	}
	reviewer, err := Get("tpl.reviewer")
	if err != nil {
		t.Fatal(err)
	}
	if len(reviewer.Memory) != 0 {
		t.Fatalf("reviewer memory=%v", reviewer.Memory)
	}
	plan, err := Get("tpl.plan-solve")
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.PlanMemory) != 1 || plan.PlanMemory[0] != "mem.retrieve" {
		t.Fatalf("plan memory=%v", plan.PlanMemory)
	}
}

func TestInjectWithoutRetrieveFailsLoad(t *testing.T) {
	_, err := Load(Manifest{
		ID: "tpl.bad", Version: Version, MaxTurns: 2, Tools: []string{"read"},
		Memory: []string{"mem.inject"},
	})
	if err == nil {
		t.Fatal("expected inject without retrieve to fail")
	}
}

func TestTemplateRejectsScenarioToolName(t *testing.T) {
	_, err := Load(Manifest{
		ID: "tpl.bad", Version: Version, MaxTurns: 2, Tools: []string{"apply_patch"},
	})
	if err == nil {
		t.Fatal("expected a non-contract tool to fail load")
	}
}

func TestLoadUsesGlobalApprovedCustomMemory(t *testing.T) {
	plugins.ResetGlobalForTest()
	t.Cleanup(plugins.ResetGlobalForTest)

	const customID = "mem.custom.retrieve"
	if err := plugins.Global().Register(plugins.Component{
		ID: customID, Kind: plugins.KindMemory,
	}); err != nil {
		t.Fatal(err)
	}
	if err := plugins.Global().Approve(customID); err != nil {
		t.Fatal(err)
	}

	got, err := Load(Manifest{
		ID: "tpl.custom-mem", Version: Version, MaxTurns: 2, Tools: []string{"read"},
		Memory: []string{customID},
	})
	if err != nil {
		t.Fatalf("Load with Global-approved custom memory: %v", err)
	}
	if len(got.Memory) != 1 || got.Memory[0] != customID {
		t.Fatalf("memory=%v", got.Memory)
	}
}

func TestLoadRejectsGlobalCandidateMemory(t *testing.T) {
	plugins.ResetGlobalForTest()
	t.Cleanup(plugins.ResetGlobalForTest)

	const customID = "mem.custom.candidate"
	if err := plugins.Global().Register(plugins.Component{
		ID: customID, Kind: plugins.KindMemory,
	}); err != nil {
		t.Fatal(err)
	}

	_, err := Load(Manifest{
		ID: "tpl.cand-mem", Version: Version, MaxTurns: 2, Tools: []string{"read"},
		Memory: []string{customID},
	})
	if err == nil {
		t.Fatal("expected Load to reject unapproved Global component")
	}
	if !errors.Is(err, plugins.ErrNotApproved) {
		t.Fatalf("want ErrNotApproved, got %v", err)
	}
}
