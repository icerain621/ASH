package session_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
	if !found["/help"] || !found["/clear"] {
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
