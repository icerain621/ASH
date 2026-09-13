package evolve_test

import (
	"testing"
	"time"

	"github.com/ash-repwiki/ash/internal/evolve"
	"github.com/ash-repwiki/ash/internal/events"
	"github.com/ash-repwiki/ash/internal/harness"
	"github.com/ash-repwiki/ash/internal/memory"
	"github.com/ash-repwiki/ash/internal/scoring"
	"github.com/ash-repwiki/ash/internal/spacepolicy"
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
		Rubric: &scoring.ReviewRubric{Correctness: 4, Safety: 4, Citable: 4, Efficiency: 4},
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

func TestMultiSignPendingSecond(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	now := time.Now().UTC()
	if err := db.Create(&store.Space{
		ID: "sp_team_ms", OrgID: "org1", Name: "Team", Kind: "team", CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	pol := spacepolicy.NewService(db)
	on := true
	if _, err := pol.PutPack("sp_team_ms", spacepolicy.PutPackRequest{MultiSign: &on}); err != nil {
		t.Fatal(err)
	}
	ev := events.NewService(db)
	har := harness.NewService(db)
	svc := evolve.NewService(db, memory.NewService(db, ev), har, nil).WithPolicy(pol)

	created, err := har.Create(harness.CreateRequest{SpaceID: "sp_team_ms", Name: "default", Spec: harness.DefaultSpec()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := har.SubmitReview(created.ID); err != nil {
		t.Fatal(err)
	}
	itemID := evolve.ItemID("harness_profile", created.ID)
	rubric := &scoring.ReviewRubric{Correctness: 4, Safety: 4, Citable: 4, Efficiency: 4}
	first, err := svc.Decide("sp_team_ms", itemID, evolve.DecideRequest{
		Decision: "approve", Reason: "first", ActorID: "alice", Rubric: rubric,
	})
	if err != nil {
		t.Fatal(err)
	}
	if first.Status != evolve.StatusPendingSecond {
		t.Fatalf("status=%s", first.Status)
	}
	second, err := svc.Decide("sp_team_ms", itemID, evolve.DecideRequest{
		Decision: "approve", Reason: "second", ActorID: "bob", Rubric: rubric,
	})
	if err != nil {
		t.Fatal(err)
	}
	if second.Status != evolve.StatusApproved {
		t.Fatalf("status=%s", second.Status)
	}
}

func TestLowScoreCreatesImproveDraft(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	ev := events.NewService(db)
	har := harness.NewService(db)
	svc := evolve.NewService(db, memory.NewService(db, ev), har, nil)

	created, err := har.Create(harness.CreateRequest{Name: "low", Spec: harness.DefaultSpec()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := har.SubmitReview(created.ID); err != nil {
		t.Fatal(err)
	}
	dec, err := svc.Decide("local", evolve.ItemID("harness_profile", created.ID), evolve.DecideRequest{
		Decision: "reject", Reason: "weak", ActorID: "rev",
		Rubric: &scoring.ReviewRubric{Correctness: 1, Safety: 1, Citable: 1, Efficiency: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	if dec.ImproveDraftID == "" {
		t.Fatal("expected improve draft")
	}
	var row store.ImproveProposal
	if err := db.First(&row, "id = ?", dec.ImproveDraftID).Error; err != nil {
		t.Fatal(err)
	}
	if row.Source != "low_score" || row.ScoreEventID == "" {
		t.Fatalf("proposal=%+v", row)
	}
}
