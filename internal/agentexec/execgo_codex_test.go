package agentexec

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecGoCodexExecutorBridgeUnavailableWhenCLIIsMissing(t *testing.T) {
	exec := &ExecGoCodexExecutor{
		CLI:        filepath.Join(t.TempDir(), "missing-execgocli"),
		CodexBin:   "/bin/echo",
		ExecGoURL:  defaultExecGoURL,
		RuntimeURL: defaultExecGoRuntimeURL,
		AgentID:    "test-agent",
	}
	_, err := exec.Execute(context.Background(), Request{RunID: "run_missing", StepID: "code.implement", RunDir: t.TempDir()})
	if !errors.Is(err, ErrBridgeUnavailable) {
		t.Fatalf("err=%v want ErrBridgeUnavailable", err)
	}
	if !strings.Contains(err.Error(), "execgocli not found") {
		t.Fatalf("err=%v want execgocli not found", err)
	}
}

func TestExecGoCodexExecutorBridgeUnavailableWhenToolsCheckFails(t *testing.T) {
	cli := writeFakeExecGoCLI(t, `#!/bin/sh
case "$1" in
  health) echo '{"ok":true,"data":{"status":"ok"}}' ;;
  tools) echo '{"ok":false,"error":{"message":"tools unavailable","status_code":503,"body":"missing runtime"}}' ;;
  *) echo '{"ok":true,"data":{}}' ;;
esac
`)
	exec := &ExecGoCodexExecutor{
		CLI:        cli,
		CodexBin:   "/bin/echo",
		ExecGoURL:  defaultExecGoURL,
		RuntimeURL: defaultExecGoRuntimeURL,
		AgentID:    "test-agent",
	}
	_, err := exec.Execute(context.Background(), Request{RunID: "run_tools", StepID: "code.implement", RunDir: t.TempDir()})
	if !errors.Is(err, ErrBridgeUnavailable) {
		t.Fatalf("err=%v want ErrBridgeUnavailable", err)
	}
	if !strings.Contains(err.Error(), "execgo tools") {
		t.Fatalf("err=%v want execgo tools context", err)
	}
}

func TestExecGoCodexExecutorSubmitsActionAndWaits(t *testing.T) {
	cli := writeFakeExecGoCLI(t, `#!/bin/sh
case "$1" in
  health) echo '{"ok":true,"data":{"status":"ok"}}' ;;
  tools) echo '{"ok":true,"data":{"schema_version":"adapter.v1","tools":["runtime.command"]}}' ;;
  act) echo '{"ok":true,"data":{"task_id":"task_codex_1"}}' ;;
  wait) echo '{"ok":true,"data":{"tasks":[{"status":"success"}]}}' ;;
  *) echo '{"ok":false,"error":{"message":"unexpected command","status_code":400,"body":"bad"}}' ;;
esac
`)
	runDir := t.TempDir()
	exec := &ExecGoCodexExecutor{
		CLI:        cli,
		CodexBin:   "/bin/echo",
		ExecGoURL:  defaultExecGoURL,
		RuntimeURL: defaultExecGoRuntimeURL,
		AgentID:    "test-agent",
	}
	res, err := exec.Execute(context.Background(), Request{
		RunID: "run_submit", TraceID: "trace_submit", StepID: "code.implement",
		RepoRoot: t.TempDir(), RunDir: runDir, Issue: "submit through execgo",
		Prompt: "make a tiny change",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.ExecGoTaskID != "task_codex_1" || res.Status != "success" {
		t.Fatalf("result=%+v want task_codex_1 success", res)
	}
	actionPath := filepath.Join(runDir, "agent-code.implement.json")
	b, err := os.ReadFile(actionPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(b)
	for _, want := range []string{`"kind": "runtime.command"`, `"program": "/bin/echo"`, `"action_id": "ash-`} {
		if !strings.Contains(text, want) {
			t.Fatalf("action JSON missing %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "--dangerously-bypass-approvals-and-sandbox") {
		t.Fatalf("action JSON should not bypass sandbox by default:\n%s", text)
	}
}

func TestExecGoCodexExecutorBypassSandboxIsExplicitOptIn(t *testing.T) {
	exec := &ExecGoCodexExecutor{BypassSandbox: true}
	args := exec.codexArgs("do work")
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "--dangerously-bypass-approvals-and-sandbox") {
		t.Fatalf("args=%v missing explicit bypass flag", args)
	}
}

func TestNewExecGoCodexExecutorReadsBypassSandboxEnv(t *testing.T) {
	t.Setenv("ASH_CODEX_BYPASS_SANDBOX", "1")
	exec := NewExecGoCodexExecutor()
	if !exec.BypassSandbox {
		t.Fatal("expected ASH_CODEX_BYPASS_SANDBOX=1 to enable bypass")
	}
}

func writeFakeExecGoCLI(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "execgocli")
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}
