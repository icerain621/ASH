package lenses

import "testing"

func TestContextPackedAdmittedOnAgentLens(t *testing.T) {
	if !Admit("context.packed", LensAgent) {
		t.Fatal("context.packed should enter the agent lens")
	}
}

func TestReplanEventsAdmittedOnAgentLens(t *testing.T) {
	for _, ev := range []string{"harness.replan_proposed", "harness.replan_exhausted"} {
		if !Admit(ev, LensAgent) {
			t.Fatalf("%s should enter the agent lens", ev)
		}
		if Admit(ev, LensMemory) {
			t.Fatalf("%s must not enter the memory lens", ev)
		}
	}
}

func TestUnregisteredEventStaysOutOfProductLenses(t *testing.T) {
	for _, lens := range []Lens{LensGlobal, LensAgent, LensMemory} {
		if Admit("not.registered", lens) {
			t.Fatalf("unregistered event entered %s", lens)
		}
	}
}
