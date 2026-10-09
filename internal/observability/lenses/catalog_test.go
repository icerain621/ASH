package lenses

import "testing"

func TestContextPackedAdmittedOnAgentLens(t *testing.T) {
	if !Admit("context.packed", LensAgent) {
		t.Fatal("context.packed should enter the agent lens")
	}
}

func TestUnregisteredEventStaysOutOfProductLenses(t *testing.T) {
	for _, lens := range []Lens{LensGlobal, LensAgent, LensMemory} {
		if Admit("not.registered", lens) {
			t.Fatalf("unregistered event entered %s", lens)
		}
	}
}
