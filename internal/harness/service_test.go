package harness_test

import (
	"testing"

	"github.com/ash-repwiki/ash/internal/harness"
	"github.com/ash-repwiki/ash/internal/store"
)

func TestValidateDefaultSpec(t *testing.T) {
	if err := harness.ValidateSpec(harness.DefaultSpec()); err != nil {
		t.Fatal(err)
	}
}

func TestValidateRejectsBadProvider(t *testing.T) {
	spec := harness.DefaultSpec()
	spec.Provider.Kind = "unknown"
	if err := harness.ValidateSpec(spec); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestCreatePromoteLoadActive(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	svc := harness.NewService(db)

	created, err := svc.Create(harness.CreateRequest{
		SpaceID:   "local",
		Name:      "default",
		Spec:      harness.DefaultSpec(),
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.Status != harness.StatusDraft {
		t.Fatalf("status=%s", created.Status)
	}

	if _, err := svc.SubmitReview(created.ID); err != nil {
		t.Fatal(err)
	}
	promoted, err := svc.Promote(created.ID, "tester")
	if err != nil {
		t.Fatal(err)
	}
	if promoted.Status != harness.StatusActive {
		t.Fatalf("status=%s want active", promoted.Status)
	}

	active, err := svc.LoadActive("local", "default")
	if err != nil {
		t.Fatal(err)
	}
	if active.ID != created.ID {
		t.Fatalf("active id=%s want %s", active.ID, created.ID)
	}

	ok, err := svc.ActiveUniquenessOK("local")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected active uniqueness")
	}
}

func TestLoadActivePlatformDefault(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	svc := harness.NewService(db)
	view, err := svc.LoadActive("local", "missing")
	if err != nil {
		t.Fatal(err)
	}
	if view.ID != "hprof_platform_default" {
		t.Fatalf("id=%s", view.ID)
	}
	if view.Spec.Provider.Kind != "execgo" {
		t.Fatalf("provider=%s", view.Spec.Provider.Kind)
	}
}

func TestUpdateNonDraftRejected(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	svc := harness.NewService(db)
	created, err := svc.Create(harness.CreateRequest{Name: "x", Spec: harness.DefaultSpec()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SubmitReview(created.ID); err != nil {
		t.Fatal(err)
	}
	spec := harness.DefaultSpec()
	if _, err := svc.Update(created.ID, harness.UpdateRequest{Spec: spec}); err == nil {
		t.Fatal("expected update rejection")
	}
}
