package evolve_test

import (
	"testing"

	"github.com/ash-repwiki/ash/internal/evolve"
	"github.com/ash-repwiki/ash/internal/events"
	"github.com/ash-repwiki/ash/internal/harness"
	"github.com/ash-repwiki/ash/internal/memory"
	"github.com/ash-repwiki/ash/internal/store"
)

func TestNormalizeTargetType(t *testing.T) {
	if _, ok := evolve.NormalizeTargetType("harness_profile"); !ok {
		t.Fatal("expected harness_profile allowed")
	}
	if _, ok := evolve.NormalizeTargetType("nope"); ok {
		t.Fatal("expected reject")
	}
}

func TestQueueAndDecideHarness(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	ev := events.NewService(db)
	mem := memory.NewService(db, ev)
	har := harness.NewService(db)
	svc := evolve.NewService(db, mem, har, nil)

	spec := harness.DefaultSpec()
	created, err := har.Create(harness.CreateRequest{Name: "default", Spec: spec})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := har.SubmitReview(created.ID); err != nil {
		t.Fatal(err)
	}

	q, err := svc.ListQueue("local", evolve.QueueOrchestration, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(q.Items) != 1 {
		t.Fatalf("items=%d", len(q.Items))
	}
	itemID := q.Items[0].ID

	dec, err := svc.Decide("local", itemID, evolve.DecideRequest{
		Decision: "approve", Reason: "looks good", ActorID: "tester",
	})
	if err != nil {
		t.Fatal(err)
	}
	if dec.Status != evolve.StatusApproved {
		t.Fatalf("status=%s", dec.Status)
	}
	active, err := har.LoadActive("local", "default")
	if err != nil {
		t.Fatal(err)
	}
	if active.ID != created.ID {
		t.Fatalf("active=%s want %s", active.ID, created.ID)
	}
}
