package agenttpl

import "testing"

func TestCompleteRejectsNilPort(t *testing.T) {
	if got := Complete(nil, "hi").Status; got != "unavailable" {
		t.Fatalf("status=%s", got)
	}
}
