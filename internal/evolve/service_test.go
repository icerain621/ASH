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

func TestAssignRoundtrip(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	ev := events.NewService(db)
	mem := memory.NewService(db, ev)
	har := harness.NewService(db)
	svc := evolve.NewService(db, mem, har, nil)

	created, err := har.Create(harness.CreateRequest{Name: "assign-me", Spec: harness.DefaultSpec()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := har.SubmitReview(created.ID); err != nil {
		t.Fatal(err)
	}
	itemID := evolve.ItemID("harness_profile", created.ID)
	if err := svc.Assign("local", itemID, "op1", "rev_alice"); err != nil {
		t.Fatal(err)
	}
	q, err := svc.ListQueue("local", evolve.QueueOrchestration, 20)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, it := range q.Items {
		if it.ID == itemID {
			found = true
			if it.AssigneeID != "rev_alice" {
				t.Fatalf("assigneeId=%q", it.AssigneeID)
			}
		}
	}
	if !found {
		t.Fatal("expected item in queue")
	}
	if err := svc.Assign("local", itemID, "op1", ""); err == nil {
		t.Fatal("empty assignee should fail")
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

func TestQueueSlaBreachFlag(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	ev := events.NewService(db)
	mem := memory.NewService(db, ev)
	har := harness.NewService(db)
	svc := evolve.NewService(db, mem, har, nil)

	oldProf, err := har.Create(harness.CreateRequest{Name: "old-sla", Spec: harness.DefaultSpec()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := har.SubmitReview(oldProf.ID); err != nil {
		t.Fatal(err)
	}
	stale := time.Now().UTC().Add(-100 * time.Hour)
	if err := db.Model(&store.HarnessProfileVersion{}).Where("id = ?", oldProf.ID).
		Update("updated_at", stale).Error; err != nil {
		t.Fatal(err)
	}

	freshProf, err := har.Create(harness.CreateRequest{Name: "fresh-sla", Spec: harness.DefaultSpec()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := har.SubmitReview(freshProf.ID); err != nil {
		t.Fatal(err)
	}

	q, err := svc.ListQueue("local", evolve.QueueOrchestration, 20)
	if err != nil {
		t.Fatal(err)
	}
	var sawOld, sawFresh bool
	for _, it := range q.Items {
		switch it.TargetID {
		case oldProf.ID:
			sawOld = true
			if !it.SlaBreach {
				t.Fatalf("old item expected slaBreach=true ageHours=%.1f", it.AgeHours)
			}
			if it.AgeHours < 72 {
				t.Fatalf("old ageHours=%.1f want >=72", it.AgeHours)
			}
		case freshProf.ID:
			sawFresh = true
			if it.SlaBreach {
				t.Fatalf("fresh item expected slaBreach=false ageHours=%.1f", it.AgeHours)
			}
		}
	}
	if !sawOld || !sawFresh {
		t.Fatalf("sawOld=%v sawFresh=%v items=%d", sawOld, sawFresh, len(q.Items))
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

func TestScoreAppealCreateListDecide(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	ev := events.NewService(db)
	mem := memory.NewService(db, ev)
	har := harness.NewService(db)
	svc := evolve.NewService(db, mem, har, nil)
	scoreSvc := scoring.NewService(db)

	score, err := scoreSvc.RecordScore("local", "memory", "mem_appeal", "", scoring.ReviewRubric{
		Correctness: 2, Safety: 2, Citable: 2, Efficiency: 2,
	}, "scorer", "weak score")
	if err != nil {
		t.Fatal(err)
	}

	item, err := svc.CreateAppeal("local", score.ID, "appealer", "please reconsider")
	if err != nil {
		t.Fatal(err)
	}
	if item.Queue != evolve.QueueAppeal || item.TargetType != evolve.TargetScoreAppeal {
		t.Fatalf("item=%+v", item)
	}
	if item.ID != evolve.ItemID(evolve.TargetScoreAppeal, score.ID) {
		t.Fatalf("id=%s", item.ID)
	}

	if _, err := svc.CreateAppeal("local", score.ID, "appealer", "dup"); err == nil {
		t.Fatal("duplicate open appeal should fail")
	}
	if _, err := svc.CreateAppeal("local", "score_missing", "appealer", "nope"); err == nil {
		t.Fatal("missing score should fail")
	}

	appealQ, err := svc.ListQueue("local", evolve.QueueAppeal, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(appealQ.Items) != 1 || appealQ.Items[0].TargetID != score.ID {
		t.Fatalf("appeal queue=%+v", appealQ.Items)
	}
	allQ, err := svc.ListQueue("local", "all", 50)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, it := range allQ.Items {
		if it.ID == item.ID {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected appeal in all queue")
	}

	if err := svc.Assign("local", item.ID, "op1", "rev_appeal"); err != nil {
		t.Fatal(err)
	}

	keep, err := svc.Decide("local", item.ID, evolve.DecideRequest{
		Decision: "approve", Reason: "keep original", ActorID: "rev",
	})
	if err != nil {
		t.Fatal(err)
	}
	if keep.Status != evolve.StatusApproved {
		t.Fatalf("keep status=%s", keep.Status)
	}
	afterKeep, err := svc.ListQueue("local", evolve.QueueAppeal, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(afterKeep.Items) != 0 {
		t.Fatalf("expected empty after keep, got %+v", afterKeep.Items)
	}

	score2, err := scoreSvc.RecordScore("local", "memory", "mem_void", "", scoring.ReviewRubric{
		Correctness: 1, Safety: 1, Citable: 1, Efficiency: 1,
	}, "scorer", "void me")
	if err != nil {
		t.Fatal(err)
	}
	voidItem, err := svc.CreateAppeal("local", score2.ID, "appealer", "void this")
	if err != nil {
		t.Fatal(err)
	}
	voided, err := svc.Decide("local", voidItem.ID, evolve.DecideRequest{
		Decision: "reject", Reason: "void score", ActorID: "rev",
	})
	if err != nil {
		t.Fatal(err)
	}
	if voided.Status != evolve.StatusRejected {
		t.Fatalf("void status=%s", voided.Status)
	}
	var voidAudit, resolvedAudit int64
	if err := db.Model(&store.AuditLog{}).Where("event_type = ? AND payload_json LIKE ?", "score.voided", "%"+score2.ID+"%").Count(&voidAudit).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&store.AuditLog{}).Where("event_type = ? AND payload_json LIKE ?", "score.appeal_resolved", "%\"decision\":\"void\"%").Count(&resolvedAudit).Error; err != nil {
		t.Fatal(err)
	}
	if voidAudit != 1 || resolvedAudit < 1 {
		t.Fatalf("voidAudit=%d resolvedAudit=%d", voidAudit, resolvedAudit)
	}
	afterVoid, err := svc.ListQueue("local", evolve.QueueAppeal, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(afterVoid.Items) != 0 {
		t.Fatalf("expected empty after void, got %+v", afterVoid.Items)
	}
}
