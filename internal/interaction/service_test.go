package interaction_test

import (
	"testing"

	"github.com/ash-repwiki/ash/internal/events"
	"github.com/ash-repwiki/ash/internal/interaction"
	"github.com/ash-repwiki/ash/internal/store"
)

func TestService_EnsureAndFoldByRun(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	ev := events.NewService(db)
	svc := interaction.NewService(db, ev)

	th, created, err := svc.EnsureThread(interaction.EnsureRequest{
		SpaceID: "local", SessionID: "sess_fold", RunID: "run_fold",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !created || th.ID == "" || th.Kind != interaction.ThreadKindMain {
		t.Fatalf("thread=%+v created=%v", th, created)
	}
	th2, created2, err := svc.EnsureThread(interaction.EnsureRequest{
		SpaceID: "local", SessionID: "sess_fold", RunID: "run_fold",
	})
	if err != nil || created2 || th2.ID != th.ID {
		t.Fatalf("idempotent failed: %+v created=%v err=%v", th2, created2, err)
	}

	if _, err := ev.Append("run_fold", "tr", "session.turn", "info", map[string]any{"prompt": "x"}); err != nil {
		t.Fatal(err)
	}
	if _, err := ev.Append("run_fold", "tr", "memory.hit_used", "info", map[string]any{
		"recordIds": []string{"mem_1"}, "count": 1,
	}); err != nil {
		t.Fatal(err)
	}

	fold, err := svc.FoldThread(th.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(fold.Nodes) < 2 || len(fold.Links) != 1 || fold.Digest == "" {
		t.Fatalf("fold=%+v", fold)
	}
	byRun, err := svc.ByRun("run_fold")
	if err != nil || byRun.Thread.ID != th.ID {
		t.Fatalf("byRun=%+v err=%v", byRun, err)
	}
}
