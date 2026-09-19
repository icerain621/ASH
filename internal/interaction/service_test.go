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

func TestService_ListThreads(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	svc := interaction.NewService(db, nil)

	th1, _, err := svc.EnsureThread(interaction.EnsureRequest{
		SpaceID: "local", SessionID: "sess_list", RunID: "run_list_a",
	})
	if err != nil {
		t.Fatal(err)
	}
	th2, _, err := svc.EnsureThread(interaction.EnsureRequest{
		SpaceID: "local", SessionID: "sess_list", RunID: "run_list_b",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.EnsureThread(interaction.EnsureRequest{
		SpaceID: "local", SessionID: "sess_other", RunID: "run_list_c",
	}); err != nil {
		t.Fatal(err)
	}

	items, err := svc.ListThreads("sess_list")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("want 2 threads, got %d: %+v", len(items), items)
	}
	if items[0].ID != th1.ID || items[1].ID != th2.ID {
		t.Fatalf("order/ids: got %+v want %s then %s", items, th1.ID, th2.ID)
	}
	if items[0].SessionID != "sess_list" || items[1].RunID != "run_list_b" {
		t.Fatalf("items=%+v", items)
	}

	empty, err := svc.ListThreads("sess_missing")
	if err != nil || len(empty) != 0 {
		t.Fatalf("empty list: %+v err=%v", empty, err)
	}
	if _, err := svc.ListThreads(""); err == nil {
		t.Fatal("expected sessionId required")
	}

	byRun, err := svc.ListThreadsByRun("run_list_a")
	if err != nil || len(byRun) != 1 || byRun[0].ID != th1.ID {
		t.Fatalf("byRun=%+v err=%v", byRun, err)
	}
}

func TestService_ForkKeepsParentAndAllowsMany(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	svc := interaction.NewService(db, events.NewService(db))
	parent, _, err := svc.EnsureThread(interaction.EnsureRequest{
		SpaceID: "local", SessionID: "sess_fork", RunID: "run_fork",
	})
	if err != nil {
		t.Fatal(err)
	}
	a, err := svc.Fork(parent.ID)
	if err != nil {
		t.Fatal(err)
	}
	b, err := svc.Fork(parent.ID)
	if err != nil {
		t.Fatal(err)
	}
	if a.ID == b.ID || a.ID == parent.ID || a.Kind != interaction.ThreadKindFork || a.ParentThreadID != parent.ID {
		t.Fatalf("fork a=%+v", a)
	}
	if b.ParentThreadID != parent.ID || b.RunID != parent.RunID || b.SessionID != parent.SessionID {
		t.Fatalf("fork b=%+v", b)
	}
	listed, err := svc.ListThreads("sess_fork")
	if err != nil || len(listed) != 3 {
		t.Fatalf("listed=%d err=%v", len(listed), err)
	}
}
