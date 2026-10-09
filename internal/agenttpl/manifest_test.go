package agenttpl

import "testing"

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
