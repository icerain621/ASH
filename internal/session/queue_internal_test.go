package session

import (
	"testing"

	"github.com/ash-repwiki/ash/internal/events"
	"github.com/ash-repwiki/ash/internal/store"
)

func TestQueue_duringFlightEnqueuesWithoutPrompt(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	svc := NewService(db, nil, events.NewService(db))
	view, err := svc.Create(CreateRequest{SpaceID: "local", CreatedBy: "actor1", RepoRoot: "."})
	if err != nil {
		t.Fatal(err)
	}
	_, flight := svc.flights.begin(view.ID)
	t.Cleanup(func() { svc.flights.end(view.ID, flight) })

	out, err := svc.Intent(view.ID, IntentRequest{Action: "queue", Prompt: "after turn"})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Turns) != 0 {
		t.Fatalf("turns=%+v", out.Turns)
	}
	got := readFollowUpQueue(out.Meta)
	if len(got) != 1 || got[0] != "after turn" {
		t.Fatalf("queue=%v", got)
	}
}
