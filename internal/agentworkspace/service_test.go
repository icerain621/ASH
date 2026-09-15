package agentworkspace

import (
	"testing"

	"github.com/ash-repwiki/ash/internal/store"
)

func TestWorkspaceLifecycle(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	svc := NewService(db)

	ws, err := svc.Create(CreateRequest{Title: "Feat ship", RepoRoot: ".", SpaceID: "local", CreatedBy: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if ws.ID == "" || ws.Status != StatusActive || len(ws.SessionIDs) != 0 {
		t.Fatalf("create=%+v", ws)
	}

	listed, err := svc.List("local", 50)
	if err != nil || len(listed) != 1 || listed[0].ID != ws.ID {
		t.Fatalf("list=%+v err=%v", listed, err)
	}

	attached, err := svc.Attach(ws.ID, "sess_a")
	if err != nil {
		t.Fatal(err)
	}
	if len(attached.SessionIDs) != 1 || attached.SessionIDs[0] != "sess_a" {
		t.Fatalf("attach=%+v", attached)
	}
	// idempotent
	again, err := svc.Attach(ws.ID, "sess_a")
	if err != nil || len(again.SessionIDs) != 1 {
		t.Fatalf("reattach=%+v err=%v", again, err)
	}

	detached, err := svc.Detach(ws.ID, "sess_a")
	if err != nil || len(detached.SessionIDs) != 0 {
		t.Fatalf("detach=%+v err=%v", detached, err)
	}
	noop, err := svc.Detach(ws.ID, "sess_missing")
	if err != nil || len(noop.SessionIDs) != 0 {
		t.Fatalf("detach missing=%+v err=%v", noop, err)
	}

	title := "Renamed WS"
	patched, err := svc.Patch(ws.ID, PatchRequest{Title: &title})
	if err != nil || patched.Title != title {
		t.Fatalf("patch=%+v err=%v", patched, err)
	}

	closed, err := svc.Close(ws.ID)
	if err != nil || closed.Status != StatusClosed {
		t.Fatalf("close=%+v err=%v", closed, err)
	}
	after, err := svc.List("local", 50)
	if err != nil || len(after) != 0 {
		t.Fatalf("list after close=%+v err=%v want empty", after, err)
	}
}

func TestFindIDBySession(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	svc := NewService(db)
	ws, err := svc.Create(CreateRequest{Title: "A", SpaceID: "local"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Attach(ws.ID, "sess_x"); err != nil {
		t.Fatal(err)
	}
	id, ok := svc.FindIDBySession("local", "sess_x")
	if !ok || id != ws.ID {
		t.Fatalf("find=%q ok=%v want %s", id, ok, ws.ID)
	}
	if _, ok := svc.FindIDBySession("local", "sess_missing"); ok {
		t.Fatal("expected miss")
	}
}
