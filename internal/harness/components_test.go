package harness_test

import (
	"strings"
	"testing"

	"github.com/ash-repwiki/ash/internal/harness"
	"github.com/ash-repwiki/ash/internal/plugins"
	"github.com/ash-repwiki/ash/internal/store"
)

func TestHarnessComponentRejectsCandidateOnPromote(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	host := plugins.NewBuiltinHost()
	if err := host.Register(plugins.Component{ID: "tool.custom.p5", Kind: plugins.KindTool, Status: plugins.StatusCandidate}); err != nil {
		t.Fatal(err)
	}
	svc := harness.NewService(db).WithHost(host)

	spec := harness.DefaultSpec()
	spec.Components = []string{"tool.custom.p5"}
	created, err := svc.Create(harness.CreateRequest{SpaceID: "local", Name: "gated", Spec: spec})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SubmitReview(created.ID); err != nil {
		t.Fatal(err)
	}
	_, err = svc.Promote(created.ID, "tester")
	if err == nil || !strings.Contains(err.Error(), "not approved") {
		t.Fatalf("err=%v", err)
	}
	if err := host.Approve("tool.custom.p5"); err != nil {
		t.Fatal(err)
	}
	promoted, err := svc.Promote(created.ID, "tester")
	if err != nil {
		t.Fatal(err)
	}
	if promoted.Status != harness.StatusActive {
		t.Fatalf("status=%s", promoted.Status)
	}
}

func TestHarnessComponentApprovedBuiltinPromote(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	host := plugins.NewBuiltinHost()
	svc := harness.NewService(db).WithHost(host)
	spec := harness.DefaultSpec()
	spec.Components = []string{"mem.retrieve", "loop.react", "compaction.threshold"}
	created, err := svc.Create(harness.CreateRequest{SpaceID: "local", Name: "builtin-comps", Spec: spec})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SubmitReview(created.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Promote(created.ID, "tester"); err != nil {
		t.Fatal(err)
	}
}
