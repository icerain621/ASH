package session

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

func TestCreateWithProviderKindACPFallsBack(t *testing.T) {
	t.Setenv("ASH_ACP_ENDPOINT", "")
	t.Setenv("ASH_ACP_URL", "")
	db := store.OpenTest(t, t.TempDir())
	svc := NewService(db, nil, events.NewService(db))
	view, err := svc.Create(CreateRequest{RepoRoot: ".", SpaceID: "local", ProviderKind: "acp_sdk", CreatedBy: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if view.ProviderKind != "acp_sdk" || !view.ProviderFallback || view.ProviderAdapter != "static" {
		t.Fatalf("view=%+v", view)
	}
	if view.Meta["providerKind"] != "acp_sdk" {
		t.Fatalf("meta=%+v", view.Meta)
	}
}

func TestPromptTurnForwardsACPWhenHealthy(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/v1/tasks", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"schema":"ash.acp.task.v1","taskId":"fwd1","status":"success","message":"ok"}`))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	t.Setenv("ASH_ACP_ENDPOINT", srv.URL)
	t.Setenv("ASH_ACP_URL", srv.URL)

	db := store.OpenTest(t, t.TempDir())
	ev := events.NewService(db)
	svc := NewService(db, nil, ev)
	now := time.Now().UTC()
	run := store.RunRecord{
		ID: "run_fwd_1", TraceID: "trace_fwd_1",
		ScenarioName: "feature_delivery", ScenarioVersion: "1.0.0",
		PolicyProfile: "default", Status: "running", SpaceID: "local",
		RepoRoot: ".", StartedAt: now, CreatedAt: now, UpdatedAt: now,
	}
	if err := db.Create(&run).Error; err != nil {
		t.Fatal(err)
	}
	view, err := svc.Create(CreateRequest{
		RunID: run.ID, SpaceID: "local", CreatedBy: "test", ProviderKind: "acp_sdk",
	})
	if err != nil {
		t.Fatal(err)
	}
	if view.ProviderFallback {
		// probe should succeed against httptest
		t.Fatalf("unexpected fallback: %+v", view)
	}
	_, _, err = svc.PromptTurn(view.ID, TurnRequest{Prompt: "forward me"})
	if err != nil {
		t.Fatal(err)
	}
	got, err := svc.Get(view.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Meta["lastAcpTaskId"] != "fwd1" {
		t.Fatalf("meta=%+v", got.Meta)
	}
	items, err := ev.ListAfter(run.ID, 0, 20)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, item := range items {
		if item.Type != "session.turn" {
			continue
		}
		var payload map[string]any
		if err := json.Unmarshal(item.Payload, &payload); err != nil {
			t.Fatal(err)
		}
		if payload["acpForwarded"] == true && payload["acpTaskId"] == "fwd1" {
			found = true
		}
	}
	if !found {
		t.Fatalf("events=%+v", items)
	}
}

func TestEnsureForRunIdempotent(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	ev := events.NewService(db)
	svc := NewService(db, nil, ev)
	now := time.Now().UTC()
	run := store.RunRecord{
		ID: "run_link_1", TraceID: "trace_link_1",
		ScenarioName: "feature_delivery", ScenarioVersion: "1.0.0",
		PolicyProfile: "default", Status: "running", SpaceID: "local",
		RepoRoot: ".", StartedAt: now, CreatedAt: now, UpdatedAt: now,
	}
	if err := db.Create(&run).Error; err != nil {
		t.Fatal(err)
	}
	bind := ProviderBinding{Kind: "acp_sdk", Adapter: "static", Fallback: true, Reason: "not configured"}
	v1, created1, err := svc.EnsureForRun("local", run.ID, ".", "test", bind)
	if err != nil || !created1 || v1 == nil {
		t.Fatalf("first=%v created=%v err=%v", v1, created1, err)
	}
	v2, created2, err := svc.EnsureForRun("local", run.ID, ".", "test", bind)
	if err != nil || created2 || v2.ID != v1.ID {
		t.Fatalf("second=%v created=%v err=%v", v2, created2, err)
	}
}

func TestListBlankSessionAndSynthesizeEvents(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	ev := events.NewService(db)
	svc := NewService(db, nil, ev)

	blank, err := svc.Create(CreateRequest{SpaceID: "local", CreatedBy: "test", RepoRoot: "."})
	if err != nil {
		t.Fatal(err)
	}
	if blank.RunID != "" {
		t.Fatalf("blank session should have empty runId: %+v", blank)
	}

	listed, err := svc.List("local", 50, false)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, item := range listed {
		if item.ID == blank.ID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("list=%+v want blank session %s", listed, blank.ID)
	}

	_, turn, err := svc.PromptTurn(blank.ID, TurnRequest{Prompt: "hello blank"})
	if err != nil {
		t.Fatal(err)
	}
	if turn == nil || turn.Prompt != "hello blank" {
		t.Fatalf("turn=%+v", turn)
	}

	evResp, err := svc.ListEvents(blank.ID, 0, 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(evResp.Items) != 1 {
		t.Fatalf("events=%+v want 1 synthesized turn", evResp.Items)
	}
	item := evResp.Items[0]
	if item.Type != "session.turn" || item.Seq != 1 || item.Visibility != events.VisibilityModelVisible {
		t.Fatalf("item=%+v", item)
	}
	var payload map[string]any
	if err := json.Unmarshal(item.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload["prompt"] != "hello blank" {
		t.Fatalf("payload=%v", payload)
	}

	// Bound-run create still works and lists alongside blank.
	now := time.Now().UTC()
	run := store.RunRecord{
		ID: "run_list_1", TraceID: "trace_list_1",
		ScenarioName: "feature_delivery", ScenarioVersion: "1.0.0",
		PolicyProfile: "default", Status: "running", SpaceID: "local",
		RepoRoot: ".", StartedAt: now, CreatedAt: now, UpdatedAt: now,
	}
	if err := db.Create(&run).Error; err != nil {
		t.Fatal(err)
	}
	bound, err := svc.Create(CreateRequest{RunID: run.ID, SpaceID: "local", CreatedBy: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if bound.RunID != run.ID {
		t.Fatalf("bound=%+v", bound)
	}
	listed, err = svc.List("local", 50, false)
	if err != nil {
		t.Fatal(err)
	}
	ids := map[string]bool{}
	for _, item := range listed {
		ids[item.ID] = true
	}
	if !ids[blank.ID] || !ids[bound.ID] {
		t.Fatalf("list ids=%v want blank+bound", ids)
	}
}

func TestPromptTurnAutoTitle(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	svc := NewService(db, nil, events.NewService(db))

	blank, err := svc.Create(CreateRequest{SpaceID: "local", CreatedBy: "test", RepoRoot: "."})
	if err != nil {
		t.Fatal(err)
	}
	if blank.Title != "" {
		t.Fatalf("blank title=%q want empty", blank.Title)
	}
	long := "一二三四五六七八九十" + "一二三四五六七八九十" + "一二三四五六七八九十" + "一二三四五六七八九十" + "尾部"
	view, _, err := svc.PromptTurn(blank.ID, TurnRequest{Prompt: long + "\nsecond line"})
	if err != nil {
		t.Fatal(err)
	}
	if view.Title == "" {
		t.Fatal("expected auto title from first prompt line")
	}
	runes := []rune(view.Title)
	if len(runes) > 48 {
		t.Fatalf("title runes=%d want <=48: %q", len(runes), view.Title)
	}
	if strings.Contains(view.Title, "\n") {
		t.Fatalf("title should be first line only: %q", view.Title)
	}
}

func TestUpdateTitleAndCloseListFilter(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	svc := NewService(db, nil, events.NewService(db))

	view, err := svc.Create(CreateRequest{SpaceID: "local", CreatedBy: "test", RepoRoot: "."})
	if err != nil {
		t.Fatal(err)
	}
	title := "My session"
	patched, err := svc.Update(view.ID, PatchRequest{Title: &title})
	if err != nil {
		t.Fatal(err)
	}
	if patched.Title != title {
		t.Fatalf("patched=%+v", patched)
	}

	closed, err := svc.Close(view.ID)
	if err != nil {
		t.Fatal(err)
	}
	if closed.Status != StatusClosed {
		t.Fatalf("status=%q", closed.Status)
	}

	active, err := svc.List("local", 50, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range active {
		if item.ID == view.ID {
			t.Fatalf("closed session should be excluded by default: %+v", active)
		}
	}
	all, err := svc.List("local", 50, true)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, item := range all {
		if item.ID == view.ID {
			found = true
			if item.Status != StatusClosed || item.Title != title {
				t.Fatalf("item=%+v", item)
			}
		}
	}
	if !found {
		t.Fatalf("includeClosed list=%+v want %s", all, view.ID)
	}

	_, _, err = svc.PromptTurn(view.ID, TurnRequest{Prompt: "nope"})
	if err == nil {
		t.Fatal("closed session must reject turns")
	}
}

func TestTruncateTitleRunes(t *testing.T) {
	got := truncateTitle("hello\nworld", 48)
	if got != "hello" {
		t.Fatalf("got=%q", got)
	}
	s := strings.Repeat("あ", 60)
	got = truncateTitle(s, 48)
	if len([]rune(got)) != 48 {
		t.Fatalf("len=%d got=%q", len([]rune(got)), got)
	}
}
