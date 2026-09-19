package session_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ash-repwiki/ash/internal/events"
	"github.com/ash-repwiki/ash/internal/session"
	"github.com/ash-repwiki/ash/internal/store"
)

func TestListCommands_builtin(t *testing.T) {
	out := session.ListCommands(".")
	if len(out.Items) < 2 {
		t.Fatalf("items=%d want >=2 builtins", len(out.Items))
	}
	found := map[string]bool{}
	for _, it := range out.Items {
		if it.Source == session.CommandSourceBuiltin {
			found[it.Name] = true
		}
		if it.Name == "" || it.Source == "" {
			t.Fatalf("item=%+v", it)
		}
	}
	if !found["/help"] || !found["/clear"] || !found["/compact"] {
		t.Fatalf("builtins missing: %v", found)
	}
}

func TestListModels_builtinKinds(t *testing.T) {
	out := session.ListModels()
	kinds := map[string]bool{}
	for _, it := range out.Items {
		kinds[it.ProviderKind] = true
		if it.ID == "" || it.Label == "" {
			t.Fatalf("item=%+v", it)
		}
	}
	for _, want := range []string{"static", "acp_sdk", "execgo"} {
		if !kinds[want] {
			t.Fatalf("missing kind %s in %+v", want, out.Items)
		}
	}
}

func TestIntent_commandHelpClearUnknown(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	ev := events.NewService(db)
	svc := session.NewService(db, nil, ev)

	view, err := svc.Create(session.CreateRequest{SpaceID: "local", CreatedBy: "actor1", RepoRoot: "."})
	if err != nil {
		t.Fatal(err)
	}
	// seed a turn so clear has something to wipe
	view, _, err = svc.PromptTurn(view.ID, session.TurnRequest{Prompt: "keep me"})
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Turns) == 0 {
		t.Fatal("expected a turn before clear")
	}

	help, err := svc.Intent(view.ID, session.IntentRequest{Action: "command", Command: "help", ActorID: "actor1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(help.Replies) == 0 || !strings.Contains(help.Replies[len(help.Replies)-1].Text, "/help") {
		t.Fatalf("help replies=%+v", help.Replies)
	}

	cleared, err := svc.Intent(view.ID, session.IntentRequest{Action: "command", Command: "/clear", ActorID: "actor1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(cleared.Turns) != 1 || cleared.Turns[0].Prompt != "/clear" {
		t.Fatalf("after clear turns=%+v", cleared.Turns)
	}
	if len(cleared.Replies) != 1 || cleared.Replies[0].Text != "已清空会话" {
		t.Fatalf("after clear replies=%+v", cleared.Replies)
	}
	if cleared.Status != session.StatusActive {
		t.Fatalf("status=%s want active", cleared.Status)
	}

	view2Run := "run_compact_1"
	now := time.Now().UTC()
	if err := db.Create(&store.RunRecord{
		ID: view2Run, TraceID: "trace_" + view2Run,
		ScenarioName: "feature_delivery", ScenarioVersion: "1.0.0",
		PolicyProfile: "default", Status: "running", SpaceID: "local",
		RepoRoot: ".", StartedAt: now, CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	view2, err := svc.Create(session.CreateRequest{RunID: view2Run, SpaceID: "local", CreatedBy: "actor1", ProviderKind: "static"})
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = svc.PromptTurn(view2.ID, session.TurnRequest{Prompt: "old a"})
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = svc.PromptTurn(view2.ID, session.TurnRequest{Prompt: "old b"})
	if err != nil {
		t.Fatal(err)
	}
	compacted, err := svc.Intent(view2.ID, session.IntentRequest{Action: "command", Command: "/compact", ActorID: "actor1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(compacted.Turns) != 1 || compacted.Turns[0].Prompt != "/compact" {
		t.Fatalf("after compact turns=%+v", compacted.Turns)
	}
	evs, err := ev.ListAfter(view2Run, 0, 50)
	if err != nil {
		t.Fatal(err)
	}
	foundCompact := false
	foundAssistant := false
	for _, e := range evs {
		if e.Type == "harness.compaction" {
			foundCompact = true
		}
		if e.Type == "assistant.message" {
			foundAssistant = true
		}
	}
	if !foundCompact || !foundAssistant {
		t.Fatalf("expected harness.compaction + assistant.message; events=%+v", evs)
	}

	_, err = svc.Intent(view.ID, session.IntentRequest{Action: "command", Command: "/nope", ActorID: "actor1"})
	if err == nil {
		t.Fatal("expected unknown command reject")
	}
	if !errors.Is(err, session.ErrIntentRejected) {
		t.Fatalf("err=%v want ErrIntentRejected", err)
	}
}

func TestUpdate_seats(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	ev := events.NewService(db)
	svc := session.NewService(db, nil, ev)

	view, err := svc.Create(session.CreateRequest{SpaceID: "local", CreatedBy: "actor1", ProviderKind: "static"})
	if err != nil {
		t.Fatal(err)
	}

	kind := "execgo"
	plan := "plan_seat_1"
	perm := session.PermissionWorkspaceWrite
	patched, err := svc.Update(view.ID, session.PatchRequest{
		ProviderKind: &kind, PlanID: &plan, PermissionMode: &perm,
	})
	if err != nil {
		t.Fatal(err)
	}
	if patched.ProviderKind != "execgo" {
		t.Fatalf("providerKind=%s", patched.ProviderKind)
	}
	if patched.PlanID != plan {
		t.Fatalf("planId=%s", patched.PlanID)
	}
	if patched.PermissionMode != session.PermissionWorkspaceWrite {
		t.Fatalf("permissionMode=%s", patched.PermissionMode)
	}

	bad := "superuser"
	_, err = svc.Update(view.ID, session.PatchRequest{PermissionMode: &bad})
	if err == nil {
		t.Fatal("expected invalid permissionMode")
	}
}

func TestUpdate_agentMode(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	ev := events.NewService(db)
	svc := session.NewService(db, nil, ev)

	view, err := svc.Create(session.CreateRequest{SpaceID: "local", CreatedBy: "actor1", ProviderKind: "static"})
	if err != nil {
		t.Fatal(err)
	}
	if view.AgentMode != session.AgentModeCoding {
		t.Fatalf("default agentMode=%q want coding", view.AgentMode)
	}

	got, err := svc.Get(view.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.AgentMode != session.AgentModeCoding {
		t.Fatalf("get agentMode=%q", got.AgentMode)
	}

	general := session.AgentModeGeneral
	patched, err := svc.Update(view.ID, session.PatchRequest{AgentMode: &general})
	if err != nil {
		t.Fatal(err)
	}
	if patched.AgentMode != session.AgentModeGeneral {
		t.Fatalf("patched agentMode=%q", patched.AgentMode)
	}
	if patched.Meta["agentMode"] != session.AgentModeGeneral {
		t.Fatalf("meta agentMode=%v", patched.Meta["agentMode"])
	}

	bad := "voice"
	_, err = svc.Update(view.ID, session.PatchRequest{AgentMode: &bad})
	if err == nil {
		t.Fatal("expected invalid agentMode")
	}
}

func TestListCommandsForSpace_includesMCP(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	now := time.Now().UTC()
	tool := store.MCPTool{
		ID: "mcp_list_1", SpaceID: "local", Name: "echo.tool",
		Server: "http://127.0.0.1:9", SchemaJSON: "{}", Risk: "medium",
		Status: "registered", CreatedAt: now, UpdatedAt: now,
	}
	if err := db.Create(&tool).Error; err != nil {
		t.Fatal(err)
	}
	disabled := store.MCPTool{
		ID: "mcp_list_off", SpaceID: "local", Name: "hidden.tool",
		Server: "http://127.0.0.1:9", SchemaJSON: "{}", Risk: "medium",
		Status: "disabled", CreatedAt: now, UpdatedAt: now,
	}
	if err := db.Create(&disabled).Error; err != nil {
		t.Fatal(err)
	}

	out := session.ListCommandsForSpace(db, "local", ".")
	found := false
	for _, it := range out.Items {
		if it.Source == session.CommandSourceMCP && it.Name == "/echo.tool" {
			found = true
			if !strings.Contains(it.Description, "MCP") {
				t.Fatalf("description=%q", it.Description)
			}
		}
		if it.Name == "/hidden.tool" {
			t.Fatalf("disabled MCP tool should be excluded")
		}
	}
	if !found {
		t.Fatalf("expected mcp /echo.tool in %+v", out.Items)
	}
}

func TestIntent_commandMCPExec(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	ev := events.NewService(db)
	svc := session.NewService(db, nil, ev)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		if req["method"] != "tools/call" {
			t.Fatalf("method=%v", req["method"])
		}
		params, _ := req["params"].(map[string]any)
		if params["name"] != "demo.echo" {
			t.Fatalf("tool name=%v", params["name"])
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"id":      req["id"],
			"result":  map[string]any{"content": []any{map[string]any{"type": "text", "text": "pong"}}},
		})
	}))
	defer srv.Close()

	now := time.Now().UTC()
	tool := store.MCPTool{
		ID: "mcp_exec_1", SpaceID: "local", Name: "demo.echo",
		Server: srv.URL, SchemaJSON: "{}", Risk: "medium",
		Status: "registered", CreatedAt: now, UpdatedAt: now,
	}
	if err := db.Create(&tool).Error; err != nil {
		t.Fatal(err)
	}

	view, err := svc.Create(session.CreateRequest{SpaceID: "local", CreatedBy: "actor1", RepoRoot: "."})
	if err != nil {
		t.Fatal(err)
	}

	out, err := svc.Intent(view.ID, session.IntentRequest{
		Action: "command", Command: "/demo.echo", Args: `{"message":"hi"}`, ActorID: "actor1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Replies) == 0 {
		t.Fatal("expected mcp reply")
	}
	reply := out.Replies[len(out.Replies)-1]
	if reply.Source != "mcp" {
		t.Fatalf("source=%q want mcp", reply.Source)
	}
	if !strings.Contains(reply.Text, "pong") {
		t.Fatalf("reply=%q want pong", reply.Text)
	}

	_, err = svc.Intent(view.ID, session.IntentRequest{
		Action: "command", Command: "/nope-mcp", ActorID: "actor1",
	})
	if err == nil {
		t.Fatal("expected unknown command reject")
	}
	if !errors.Is(err, session.ErrIntentRejected) {
		t.Fatalf("err=%v want ErrIntentRejected", err)
	}
}

func TestIntent_commandSkillExec(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	ev := events.NewService(db)
	svc := session.NewService(db, nil, ev)

	repo := t.TempDir()
	skillDir := filepath.Join(repo, ".ash", "skills", "demo-skill")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "---\nname: demo-skill\ndescription: Demo skill body.\n---\n\n# Demo Skill\n\nFollow these steps carefully.\n"
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	view, err := svc.Create(session.CreateRequest{
		SpaceID: "local", CreatedBy: "actor1", RepoRoot: repo,
	})
	if err != nil {
		t.Fatal(err)
	}

	out, err := svc.Intent(view.ID, session.IntentRequest{
		Action: "command", Command: "/demo-skill", Args: "focus on tests", ActorID: "actor1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Replies) == 0 {
		t.Fatal("expected skill reply")
	}
	reply := out.Replies[len(out.Replies)-1]
	if reply.Source != "skill" {
		t.Fatalf("source=%q want skill", reply.Source)
	}
	if !strings.Contains(reply.Text, "Demo Skill") && !strings.Contains(reply.Text, "demo-skill") {
		t.Fatalf("reply=%q want skill title", reply.Text)
	}
	if !strings.Contains(reply.Text, "已加载技能") {
		t.Fatalf("reply=%q want loaded hint", reply.Text)
	}
}
