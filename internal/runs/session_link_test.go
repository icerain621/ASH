package runs

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/ash-repwiki/ash/internal/events"
	"github.com/ash-repwiki/ash/internal/harness"
	"github.com/ash-repwiki/ash/internal/rules"
	"github.com/ash-repwiki/ash/internal/store"
	"github.com/ash-repwiki/ash/internal/toolbus"
)

type fakeSessionLinker struct {
	lastBind SessionProviderBind
	lastRun  string
	id       string
}

func (f *fakeSessionLinker) EnsureForRun(spaceID, runID, repoRoot, createdBy string, bind SessionProviderBind) (string, bool, error) {
	f.lastRun = runID
	f.lastBind = bind
	if f.id == "" {
		f.id = "sess_fake_" + runID
	}
	return f.id, true, nil
}

func (f *fakeSessionLinker) WithContext(ctx context.Context) SessionLinker { return f }

func TestLinkProviderSessionCreatesDocument(t *testing.T) {
	t.Setenv("ASH_ACP_ENDPOINT", "")
	dir := t.TempDir()
	db, err := store.Open(filepath.Join(dir, "ash.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	ev := events.NewService(db)
	fake := &fakeSessionLinker{}
	svc := NewService(db, ev, rules.NewLoader("scenarios"), toolbus.DefaultBus()).WithSessionService(fake)

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

	now := time.Now().UTC()
	run := store.RunRecord{
		ID: "run_acp_link", TraceID: "trace_acp_link",
		ScenarioName: "feature_delivery", ScenarioVersion: "1.0.0",
		PolicyProfile: "default", Status: "running", SpaceID: "local",
		RepoRoot: ".", StartedAt: now, CreatedAt: now, UpdatedAt: now,
	}
	if err := db.Create(&run).Error; err != nil {
		t.Fatal(err)
	}

	sel := svc.SelectProvider("local")
	if sel.RequestedKind != "acp_sdk" || !sel.Fallback {
		t.Fatalf("sel=%+v", sel)
	}
	sessionID := svc.linkProviderSession("local", run.ID, run.TraceID, ".", sel)
	if sessionID == "" {
		t.Fatal("expected session id")
	}
	if fake.lastRun != run.ID || fake.lastBind.Kind != "acp_sdk" {
		t.Fatalf("fake lastRun=%q bind=%+v", fake.lastRun, fake.lastBind)
	}
	items, err := ev.ListAfter(run.ID, 0, 20)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, item := range items {
		if item.Type == "session.linked" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("events=%+v want session.linked", items)
	}
}
