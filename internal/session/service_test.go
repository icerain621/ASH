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
	if view.RunID != run.ID || view.StreamURL == "" || !strings.Contains(view.StreamURL, view.ID) {
		t.Fatalf("view=%+v", view)
	}
	if !strings.Contains(view.StreamURL, "/agents/sessions/") {
		t.Fatalf("streamUrl=%q want session stream path", view.StreamURL)
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
	foundTurn, foundMsg := false, false
	for _, item := range items {
		switch item.Type {
		case "session.turn":
			foundTurn = true
		case "assistant.message":
			foundMsg = true
			var payload map[string]any
			if err := json.Unmarshal(item.Payload, &payload); err != nil {
				t.Fatal(err)
			}
			if payload["source"] != "echo" || payload["text"] != "已收到：continue with tests" {
				t.Fatalf("assistant.message payload=%v", payload)
			}
		}
	}
	if !foundTurn || !foundMsg {
		t.Fatalf("events=%+v want session.turn + assistant.message", items)
	}

	got, err := svc.Get(view.ID)
	if err != nil || got.ID != view.ID {
		t.Fatalf("get=%+v err=%v", got, err)
	}
}

func TestBlankPromptTurnSynthesizesAssistantStream(t *testing.T) {
	t.Setenv("ASH_LLM_BASE_URL", "")
	db := store.OpenTest(t, t.TempDir())
	svc := NewService(db, nil, events.NewService(db))

	blank, err := svc.Create(CreateRequest{SpaceID: "local", CreatedBy: "test", RepoRoot: "."})
	if err != nil {
		t.Fatal(err)
	}
	view, turn, err := svc.PromptTurn(blank.ID, TurnRequest{Prompt: "ping"})
	if err != nil {
		t.Fatal(err)
	}
	if turn == nil || len(view.Replies) != 1 {
		t.Fatalf("view=%+v turn=%+v", view, turn)
	}
	if view.Replies[0].Source != "echo" || view.Replies[0].Text != "已收到：ping" {
		t.Fatalf("reply=%+v", view.Replies[0])
	}
	if n := len(view.Replies[0].Chunks); n < 2 || n > 4 {
		t.Fatalf("chunks=%d want 2–4", n)
	}

	evResp, err := svc.ListEvents(blank.ID, 0, 50)
	if err != nil {
		t.Fatal(err)
	}
	types := make([]string, 0, len(evResp.Items))
	for _, item := range evResp.Items {
		types = append(types, item.Type)
	}
	if len(types) < 3 || types[0] != "session.turn" {
		t.Fatalf("types=%v", types)
	}
	sawDelta, sawMsg := false, false
	for _, typ := range types[1:] {
		switch typ {
		case "assistant.delta":
			sawDelta = true
		case "assistant.message":
			sawMsg = true
		}
	}
	if !sawDelta || !sawMsg {
		t.Fatalf("types=%v want deltas + message after turn", types)
	}
}

func TestBoundPromptTurnAppendsAssistantMessage(t *testing.T) {
	t.Setenv("ASH_LLM_BASE_URL", "")
	db := store.OpenTest(t, t.TempDir())
	ev := events.NewService(db)
	svc := NewService(db, nil, ev)
	now := time.Now().UTC()
	run := store.RunRecord{
		ID: "run_asst_1", TraceID: "trace_asst_1",
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
	_, _, err = svc.PromptTurn(view.ID, TurnRequest{Prompt: "bound hello"})
	if err != nil {
		t.Fatal(err)
	}
	resp, err := svc.ListEvents(view.ID, 0, 50)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, item := range resp.Items {
		if item.Type == "assistant.message" {
			found = true
			var payload map[string]any
			if err := json.Unmarshal(item.Payload, &payload); err != nil {
				t.Fatal(err)
			}
			if payload["text"] != "已收到：bound hello" || payload["source"] != "echo" {
				t.Fatalf("payload=%v", payload)
			}
			if payload["stopped"] != false {
				t.Fatalf("stopped=%v", payload["stopped"])
			}
		}
	}
	if !found {
		t.Fatalf("events=%+v want assistant.message", resp.Items)
	}
}

func TestSplitReplyChunksRange(t *testing.T) {
	chunks := splitReplyChunks("short")
	if len(chunks) < 2 || len(chunks) > 4 {
		t.Fatalf("chunks=%v", chunks)
	}
	joined := strings.Join(chunks, "")
	if joined != "short" {
		t.Fatalf("joined=%q", joined)
	}
	long := strings.Repeat("字", 100)
	chunks = splitReplyChunks(long)
	if len(chunks) != 4 {
		t.Fatalf("long chunks=%d want 4", len(chunks))
	}
	if strings.Join(chunks, "") != long {
		t.Fatal("long join mismatch")
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
	if len(evResp.Items) < 2 {
		t.Fatalf("events=%+v want session.turn + assistant.message (and deltas)", evResp.Items)
	}
	var sawTurn, sawMsg, sawDelta bool
	for _, item := range evResp.Items {
		switch item.Type {
		case "session.turn":
			sawTurn = true
			if item.Seq != 1 || item.Visibility != events.VisibilityModelVisible {
				t.Fatalf("turn item=%+v", item)
			}
			var payload map[string]any
			if err := json.Unmarshal(item.Payload, &payload); err != nil {
				t.Fatal(err)
			}
			if payload["prompt"] != "hello blank" {
				t.Fatalf("payload=%v", payload)
			}
		case "assistant.delta":
			sawDelta = true
		case "assistant.message":
			sawMsg = true
			var payload map[string]any
			if err := json.Unmarshal(item.Payload, &payload); err != nil {
				t.Fatal(err)
			}
			if payload["source"] != "echo" || payload["text"] != "已收到：hello blank" {
				t.Fatalf("assistant.message payload=%v", payload)
			}
		}
	}
	if !sawTurn || !sawMsg || !sawDelta {
		t.Fatalf("events=%+v want turn+delta+message", evResp.Items)
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

	purged, err := svc.Purge(view.ID)
	if err != nil {
		t.Fatal(err)
	}
	if purged == nil || !purged.Purged || purged.ID != view.ID {
		t.Fatalf("purge=%+v", purged)
	}
	if _, err := svc.Get(view.ID); err == nil {
		t.Fatal("expected Get after purge to fail")
	}
	allAfterPurge, err := svc.List("local", 50, true)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range allAfterPurge {
		if item.ID == view.ID {
			t.Fatalf("purged session still listed: %+v", allAfterPurge)
		}
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

func TestPromptTurnLLMStreamBlank(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"LL\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"M ok\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	t.Cleanup(srv.Close)
	t.Setenv("ASH_LLM_BASE_URL", srv.URL)
	t.Setenv("ASH_LLM_API_KEY", "sk-test")
	t.Setenv("ASH_LLM_MODEL", "gpt-4o-mini")

	db := store.OpenTest(t, t.TempDir())
	svc := NewService(db, nil, events.NewService(db))
	blank, err := svc.Create(CreateRequest{SpaceID: "local", CreatedBy: "test", RepoRoot: "."})
	if err != nil {
		t.Fatal(err)
	}
	view, _, err := svc.PromptTurn(blank.ID, TurnRequest{Prompt: "ping llm"})
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Replies) != 1 || view.Replies[0].Source != "llm" || view.Replies[0].Text != "LLM ok" {
		t.Fatalf("reply=%+v", view.Replies)
	}
	evResp, err := svc.ListEvents(blank.ID, 0, 50)
	if err != nil {
		t.Fatal(err)
	}
	sawDelta, sawMsg := false, false
	for _, item := range evResp.Items {
		switch item.Type {
		case "assistant.delta":
			sawDelta = true
		case "assistant.message":
			sawMsg = true
			var payload map[string]any
			if err := json.Unmarshal(item.Payload, &payload); err != nil {
				t.Fatal(err)
			}
			if payload["source"] != "llm" || payload["text"] != "LLM ok" {
				t.Fatalf("payload=%v", payload)
			}
		}
	}
	if !sawDelta || !sawMsg {
		t.Fatalf("events=%+v", evResp.Items)
	}
}

func TestPromptTurnProviderStatic(t *testing.T) {
	t.Setenv("ASH_LLM_BASE_URL", "")
	db := store.OpenTest(t, t.TempDir())
	svc := NewService(db, nil, events.NewService(db))
	blank, err := svc.Create(CreateRequest{
		SpaceID: "local", CreatedBy: "test", RepoRoot: ".", ProviderKind: "static",
	})
	if err != nil {
		t.Fatal(err)
	}
	if blank.ProviderKind != "static" || blank.ProviderAdapter != "static" {
		t.Fatalf("view=%+v", blank)
	}
	view, _, err := svc.PromptTurn(blank.ID, TurnRequest{Prompt: "run static"})
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Replies) != 1 {
		t.Fatalf("replies=%+v", view.Replies)
	}
	if view.Replies[0].Source != "static" {
		t.Fatalf("source=%q want static", view.Replies[0].Source)
	}
	if !strings.Contains(view.Replies[0].Text, "static") {
		t.Fatalf("text=%q", view.Replies[0].Text)
	}
	evResp, err := svc.ListEvents(blank.ID, 0, 50)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, item := range evResp.Items {
		if item.Type != "assistant.message" {
			continue
		}
		var payload map[string]any
		if err := json.Unmarshal(item.Payload, &payload); err != nil {
			t.Fatal(err)
		}
		if payload["source"] == "static" {
			found = true
		}
	}
	if !found {
		t.Fatalf("events=%+v want source=static", evResp.Items)
	}
}
