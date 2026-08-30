package runs

import (
	"path/filepath"
	"testing"

	"github.com/ash-repwiki/ash/internal/agentexec"
	"github.com/ash-repwiki/ash/internal/events"
	"github.com/ash-repwiki/ash/internal/harness"
	"github.com/ash-repwiki/ash/internal/rules"
	"github.com/ash-repwiki/ash/internal/store"
	"github.com/ash-repwiki/ash/internal/toolbus"
)

func TestSelectProviderPinned(t *testing.T) {
	dir := t.TempDir()
	db, err := store.Open(filepath.Join(dir, "ash.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	svc := NewService(db, events.NewService(db), rules.NewLoader("scenarios"), toolbus.DefaultBus()).
		WithAgentExecutor(agentexec.StaticExecutor{})
	sel := svc.SelectProvider("local")
	if sel.Source != "pinned" || sel.Adapter != "static" || sel.Fallback {
		t.Fatalf("sel=%+v", sel)
	}
}

func TestSelectProviderHarnessStatic(t *testing.T) {
	dir := t.TempDir()
	db, err := store.Open(filepath.Join(dir, "ash.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	svc := NewService(db, events.NewService(db), rules.NewLoader("scenarios"), toolbus.DefaultBus())
	spec := harness.DefaultSpec()
	spec.Provider.Kind = "static"
	created, err := svc.harnessSvc.Create(harness.CreateRequest{Name: "default", Spec: spec, CreatedBy: "tester"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.harnessSvc.SubmitReview(created.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.harnessSvc.Promote(created.ID, "tester"); err != nil {
		t.Fatal(err)
	}
	sel := svc.SelectProvider("local")
	if sel.Source != "harness" || sel.RequestedKind != "static" || sel.Adapter != "static" {
		t.Fatalf("sel=%+v", sel)
	}
}

func TestSelectProviderHarnessACPFallsBack(t *testing.T) {
	t.Setenv("ASH_ACP_ENDPOINT", "")
	t.Setenv("ASH_ACP_URL", "")
	dir := t.TempDir()
	db, err := store.Open(filepath.Join(dir, "ash.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	svc := NewService(db, events.NewService(db), rules.NewLoader("scenarios"), toolbus.DefaultBus())
	spec := harness.DefaultSpec()
	spec.Provider.Kind = "acp_sdk"
	created, err := svc.harnessSvc.Create(harness.CreateRequest{Name: "default", Spec: spec, CreatedBy: "tester"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.harnessSvc.SubmitReview(created.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.harnessSvc.Promote(created.ID, "tester"); err != nil {
		t.Fatal(err)
	}
	sel := svc.SelectProvider("local")
	if sel.Source != "harness" || sel.RequestedKind != "acp_sdk" || !sel.Fallback || sel.Adapter != "static" {
		t.Fatalf("sel=%+v", sel)
	}
}
