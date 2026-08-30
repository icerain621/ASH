package session

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/ash-repwiki/ash/internal/events"
	"github.com/ash-repwiki/ash/internal/store"
)

func TestCreateBindRunAndPromptTurn(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	ev := events.NewService(db)
	svc := NewService(db, nil, ev)
	now := time.Now().UTC()
	run := store.RunRecord{
		ID: "run_sess_1", TraceID: "trace_sess_1",
		ScenarioName: "feature_delivery", ScenarioVersion: "1.0.0",
		PolicyProfile: "default", Status: "running", SpaceID: "local",
		RepoRoot: ".", StartedAt: now, CreatedAt: now, UpdatedAt: now,
	}
	if err := db.Create(&run).Error; err != nil {
		t.Fatal(err)
	}

	view, err := svc.Create(CreateRequest{RunID: run.ID, SpaceID: "local", CreatedBy: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if view.RunID != run.ID || view.StreamURL == "" || !strings.Contains(view.StreamURL, run.ID) {
		t.Fatalf("view=%+v", view)
	}

	view2, turn, err := svc.PromptTurn(view.ID, TurnRequest{Prompt: "continue with tests"})
	if err != nil {
		t.Fatal(err)
	}
	if turn == nil || turn.Prompt == "" || len(view2.Turns) != 1 {
		t.Fatalf("turn=%+v view=%+v", turn, view2)
	}
	items, err := ev.ListAfter(run.ID, 0, 20)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, item := range items {
		if item.Type == "session.turn" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("events=%+v want session.turn", items)
	}

	got, err := svc.Get(view.ID)
	if err != nil || got.ID != view.ID {
		t.Fatalf("get=%+v err=%v", got, err)
	}
}

func TestServeRPCSessionStartIdle(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	svc := NewService(db, nil, events.NewService(db))
	in := bytes.NewBufferString(`{"type":"session.start","repoRoot":"."}` + "\n")
	var out bytes.Buffer
	if err := svc.ServeRPC(in, &out); err != nil {
		t.Fatal(err)
	}
	var ev RPCEvent
	if err := json.Unmarshal(out.Bytes(), &ev); err != nil {
		t.Fatalf("out=%q err=%v", out.String(), err)
	}
	if ev.Name != "session.started" || ev.SessionID == "" {
		t.Fatalf("ev=%+v", ev)
	}
}

func TestCreateRejectsGoalAndRunTogether(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	svc := NewService(db, nil, events.NewService(db))
	_, err := svc.Create(CreateRequest{Goal: "x", RunID: "run_x"})
	if err == nil {
		t.Fatal("expected error")
	}
}
