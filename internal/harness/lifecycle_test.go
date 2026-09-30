package harness_test

import (
	"testing"

	"github.com/ash-repwiki/ash/internal/harness"
	"github.com/ash-repwiki/ash/internal/store"
)

func TestRejectAndPendingSecond(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	svc := harness.NewService(db)

	created, err := svc.Create(harness.CreateRequest{Name: "review-flow", Spec: harness.DefaultSpec()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SubmitReview(created.ID); err != nil {
		t.Fatal(err)
	}
	pending, err := svc.MarkPendingSecond(created.ID, "approver-1")
	if err != nil {
		t.Fatal(err)
	}
	if pending.Status != harness.StatusPendingSecond {
		t.Fatalf("status=%s", pending.Status)
	}
	rejected, err := svc.Reject(created.ID, "approver-1", "nogo")
	if err != nil {
		t.Fatal(err)
	}
	if rejected.Status != harness.StatusArchived {
		t.Fatalf("status=%s want archived", rejected.Status)
	}
}

func TestPromoteThenRollback(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	svc := harness.NewService(db)

	v1, err := svc.Create(harness.CreateRequest{Name: "rb", Spec: harness.DefaultSpec(), CreatedBy: "a"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SubmitReview(v1.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Promote(v1.ID, "a"); err != nil {
		t.Fatal(err)
	}

	v2, err := svc.Create(harness.CreateRequest{Name: "rb", Spec: harness.DefaultSpec(), CreatedBy: "b"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SubmitReview(v2.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Promote(v2.ID, "b"); err != nil {
		t.Fatal(err)
	}

	rolled, err := svc.Rollback(v2.ID, "ops")
	if err != nil {
		t.Fatal(err)
	}
	if rolled.ID != v1.ID {
		t.Fatalf("rollback id=%s want %s", rolled.ID, v1.ID)
	}
	if rolled.Status != harness.StatusActive {
		t.Fatalf("status=%s", rolled.Status)
	}
}

func TestListGetAndCreateValidation(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	svc := harness.NewService(db)

	if _, err := svc.Create(harness.CreateRequest{Spec: harness.DefaultSpec()}); err == nil {
		t.Fatal("empty name must fail")
	}
	created, err := svc.Create(harness.CreateRequest{SpaceID: "sp1", Name: "listed", Spec: harness.DefaultSpec()})
	if err != nil {
		t.Fatal(err)
	}
	got, err := svc.Get(created.ID)
	if err != nil || got.ID != created.ID {
		t.Fatalf("get: %v %+v", err, got)
	}
	if _, err := svc.Get("missing"); err == nil {
		t.Fatal("want get error")
	}
	list, err := svc.List("sp1", harness.StatusDraft, "listed")
	if err != nil || len(list) != 1 {
		t.Fatalf("list: %v n=%d", err, len(list))
	}
}

func TestMarkPendingSecondWrongStatus(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	svc := harness.NewService(db)
	created, err := svc.Create(harness.CreateRequest{Name: "draft-only", Spec: harness.DefaultSpec()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.MarkPendingSecond(created.ID, "x"); err == nil {
		t.Fatal("draft must not pending_second")
	}
	if _, err := svc.Rollback(created.ID, "x"); err == nil {
		t.Fatal("non-active must not rollback")
	}
}
