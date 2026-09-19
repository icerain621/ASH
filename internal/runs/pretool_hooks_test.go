package runs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ash-repwiki/ash/internal/agentexec"
	"github.com/ash-repwiki/ash/internal/events"
	"github.com/ash-repwiki/ash/internal/rules"
	"github.com/ash-repwiki/ash/internal/store"
	"github.com/ash-repwiki/ash/internal/toolbus"
)

func writeVerifyEchoScenario(t *testing.T, scenariosDir, name string) {
	t.Helper()
	if err := os.MkdirAll(scenariosDir, 0o755); err != nil {
		t.Fatal(err)
	}
	scenario := `version: "ash.rules/v0.1"
scenario:
  name: "` + name + `"
  scenarioVersion: "1.0.0"
  roles:
    Admin: { maxParallel: 1 }
  inputs:
    required: [issueOrSpec]
  steps:
    - id: "qa.verify"
      role: "Admin"
      kind: "verify"
      verify:
        checks:
          - tool: "echo.safe"
`
	if err := os.WriteFile(filepath.Join(scenariosDir, name+".yaml"), []byte(scenario), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeSafeEchoScenario(t *testing.T, scenariosDir, name string) {
	t.Helper()
	if err := os.MkdirAll(scenariosDir, 0o755); err != nil {
		t.Fatal(err)
	}
	scenario := `version: "ash.rules/v0.1"
scenario:
  name: "` + name + `"
  scenarioVersion: "1.0.0"
  roles:
    Admin: { maxParallel: 1 }
  inputs:
    required: [issueOrSpec]
  steps:
    - id: "ops.echo"
      role: "Admin"
      kind: "tool_chain"
      chain:
        - tool: "echo.safe"
`
	if err := os.WriteFile(filepath.Join(scenariosDir, name+".yaml"), []byte(scenario), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestPreToolUseHookDenySkipsTool(t *testing.T) {
	dir := t.TempDir()
	db := store.OpenTest(t, dir)
	now := time.Now().UTC()
	body := `{"hooks":{"version":"ash.hooks.v1","rules":[{"event":"PreToolUse","tool":"echo.safe","action":"deny","reason":"blocked by space hook"}]}}`
	if err := db.Create(&store.SpacePolicyPack{
		SpaceID: "local", CitationMode: "optional", BodyJSON: body,
		CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	scenariosDir := filepath.Join(dir, "scenarios")
	writeSafeEchoScenario(t, scenariosDir, "hook_deny")
	loader := rules.NewLoader(scenariosDir)
	if err := loader.LoadDir(); err != nil {
		t.Fatal(err)
	}

	calls := 0
	reg := toolbus.NewRegistry()
	reg.Register("echo.safe", toolbus.RiskSafe, func(_ toolbus.Context, _ map[string]any) (map[string]any, error) {
		calls++
		return map[string]any{"ok": true}, nil
	})
	ev := events.NewService(db)
	svc := NewService(db, ev, loader, toolbus.NewBus(reg)).WithAgentExecutor(agentexec.StaticExecutor{})

	created, err := svc.Create(CreateRequest{
		Scenario: ScenarioRef{Name: "hook_deny", ScenarioVersion: "1.0.0"},
		Inputs:   map[string]any{"issueOrSpec": "hook deny"},
	})
	if created == nil || created.RunID == "" {
		t.Fatal(err)
	}
	if calls != 0 {
		t.Fatalf("tool executed despite deny: calls=%d", calls)
	}
	if err == nil || !strings.Contains(err.Error(), "HOOK_DENIED") {
		t.Fatalf("err=%v want HOOK_DENIED", err)
	}
	sum, err := svc.Get(created.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if sum.Status != "failed" {
		t.Fatalf("status=%q want failed", sum.Status)
	}
	var step store.RunStep
	if err := db.Where("run_id = ?", created.RunID).Order("created_at asc").First(&step).Error; err != nil {
		t.Fatal(err)
	}
	if step.ErrorCode != "HOOK_DENIED" {
		t.Fatalf("errorCode=%q want HOOK_DENIED", step.ErrorCode)
	}

	evs, err := ev.ListAfter(created.RunID, 0, 100)
	if err != nil {
		t.Fatal(err)
	}
	foundPre, foundDecision := false, false
	for _, item := range evs {
		switch item.Type {
		case "hook.pre_tool_use":
			foundPre = true
			if !strings.Contains(string(item.Payload), "deny") {
				t.Fatalf("hook.pre_tool_use payload=%s want deny", item.Payload)
			}
		case "hook.decision":
			foundDecision = true
		}
	}
	if !foundPre || !foundDecision {
		t.Fatalf("events missing hook.pre_tool_use=%v hook.decision=%v", foundPre, foundDecision)
	}
}

func TestPreToolUseHookDenyVerifyStep(t *testing.T) {
	dir := t.TempDir()
	db := store.OpenTest(t, dir)
	now := time.Now().UTC()
	body := `{"hooks":{"version":"ash.hooks.v1","rules":[{"event":"PreToolUse","tool":"echo.safe","action":"deny","reason":"blocked by space hook"}]}}`
	if err := db.Create(&store.SpacePolicyPack{
		SpaceID: "local", CitationMode: "optional", BodyJSON: body,
		CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	scenariosDir := filepath.Join(dir, "scenarios")
	writeVerifyEchoScenario(t, scenariosDir, "hook_deny_verify")
	loader := rules.NewLoader(scenariosDir)
	if err := loader.LoadDir(); err != nil {
		t.Fatal(err)
	}

	calls := 0
	reg := toolbus.NewRegistry()
	reg.Register("echo.safe", toolbus.RiskSafe, func(_ toolbus.Context, _ map[string]any) (map[string]any, error) {
		calls++
		return map[string]any{"ok": true}, nil
	})
	ev := events.NewService(db)
	svc := NewService(db, ev, loader, toolbus.NewBus(reg)).WithAgentExecutor(agentexec.StaticExecutor{})

	created, err := svc.Create(CreateRequest{
		Scenario: ScenarioRef{Name: "hook_deny_verify", ScenarioVersion: "1.0.0"},
		Inputs:   map[string]any{"issueOrSpec": "verify hook deny"},
	})
	if created == nil || created.RunID == "" {
		t.Fatal(err)
	}
	if calls != 0 {
		t.Fatalf("verify tool executed despite deny: calls=%d", calls)
	}
	if err == nil || !strings.Contains(err.Error(), "HOOK_DENIED") {
		t.Fatalf("err=%v want HOOK_DENIED", err)
	}
	var step store.RunStep
	if err := db.Where("run_id = ?", created.RunID).First(&step).Error; err != nil {
		t.Fatal(err)
	}
	if step.ErrorCode != "HOOK_DENIED" {
		t.Fatalf("errorCode=%q want HOOK_DENIED", step.ErrorCode)
	}
}

func TestPreToolUseNoConfigUnchanged(t *testing.T) {
	dir := t.TempDir()
	db := store.OpenTest(t, dir)
	scenariosDir := filepath.Join(dir, "scenarios")
	writeSafeEchoScenario(t, scenariosDir, "hook_none")
	loader := rules.NewLoader(scenariosDir)
	if err := loader.LoadDir(); err != nil {
		t.Fatal(err)
	}

	calls := 0
	reg := toolbus.NewRegistry()
	reg.Register("echo.safe", toolbus.RiskSafe, func(_ toolbus.Context, _ map[string]any) (map[string]any, error) {
		calls++
		return map[string]any{"ok": true}, nil
	})
	ev := events.NewService(db)
	svc := NewService(db, ev, loader, toolbus.NewBus(reg)).WithAgentExecutor(agentexec.StaticExecutor{})

	created, err := svc.Create(CreateRequest{
		Scenario: ScenarioRef{Name: "hook_none", ScenarioVersion: "1.0.0"},
		Inputs:   map[string]any{"issueOrSpec": "no hooks"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("calls=%d want 1", calls)
	}
	sum, err := svc.Get(created.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if sum.Status != "finished" {
		t.Fatalf("status=%q want finished", sum.Status)
	}
	evs, err := ev.ListAfter(created.RunID, 0, 100)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range evs {
		if item.Type == "hook.pre_tool_use" || item.Type == "hook.decision" {
			t.Fatalf("unexpected hook event %s without hooks config", item.Type)
		}
	}
}

func TestPreToolUseHookAskRequiresApproval(t *testing.T) {
	dir := t.TempDir()
	db := store.OpenTest(t, dir)
	now := time.Now().UTC()
	body := `{"hooks":{"version":"ash.hooks.v1","rules":[{"event":"PreToolUse","tool":"echo.safe","action":"ask","reason":"confirm echo"}]}}`
	if err := db.Create(&store.SpacePolicyPack{
		SpaceID: "local", CitationMode: "optional", BodyJSON: body,
		CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	scenariosDir := filepath.Join(dir, "scenarios")
	writeSafeEchoScenario(t, scenariosDir, "hook_ask")
	loader := rules.NewLoader(scenariosDir)
	if err := loader.LoadDir(); err != nil {
		t.Fatal(err)
	}

	calls := 0
	reg := toolbus.NewRegistry()
	reg.Register("echo.safe", toolbus.RiskSafe, func(_ toolbus.Context, _ map[string]any) (map[string]any, error) {
		calls++
		return map[string]any{"ok": true}, nil
	})
	ev := events.NewService(db)
	svc := NewService(db, ev, loader, toolbus.NewBus(reg)).WithAgentExecutor(agentexec.StaticExecutor{})

	created, err := svc.Create(CreateRequest{
		Scenario: ScenarioRef{Name: "hook_ask", ScenarioVersion: "1.0.0"},
		Inputs:   map[string]any{"issueOrSpec": "hook ask"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 0 {
		t.Fatalf("tool ran before approval: calls=%d", calls)
	}
	sum, err := svc.Get(created.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if sum.Status != "waiting_approval" {
		t.Fatalf("status=%q want waiting_approval", sum.Status)
	}
	var step store.RunStep
	if err := db.Where("run_id = ?", created.RunID).First(&step).Error; err != nil {
		t.Fatal(err)
	}
	if step.ErrorCode != "HOOK_ASK_APPROVAL_REQUIRED" {
		t.Fatalf("errorCode=%q want HOOK_ASK_APPROVAL_REQUIRED", step.ErrorCode)
	}

	if _, err := svc.Approve(created.RunID, ApproveRequest{ActorID: "tester", Reason: "allow hook"}); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("calls=%d want 1 after approval", calls)
	}
	sum, err = svc.Get(created.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if sum.Status != "finished" {
		t.Fatalf("status=%q want finished", sum.Status)
	}
}
