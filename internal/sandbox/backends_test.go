package sandbox

import (
	"slices"
	"testing"
)

func TestKnownBackendIDs(t *testing.T) {
	got := KnownBackendIDs()
	want := []string{"local", "landlock", "docker", "remote-mock", "remote-e2b"}
	if !slices.Equal(got, want) {
		t.Fatalf("got=%v want=%v", got, want)
	}
	got[0] = "mutated"
	if KnownBackendIDs()[0] != "local" {
		t.Fatal("KnownBackendIDs must return a fresh slice")
	}
}
