package toolbus

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ash-repwiki/ash/internal/testutil"
)

func TestRuntimeCommandSubmitsThroughExecGoCLI(t *testing.T) {
	cli := writeFakeRuntimeExecGoCLI(t, `#!/bin/sh
case "$1" in
  health) echo '{"ok":true,"data":{"status":"ok"}}' ;;
  tools) echo '{"ok":true,"data":{"schema_version":"adapter.v1","tools":["runtime.command"]}}' ;;
  act) echo '{"ok":true,"data":{"task_id":"task_runtime_1"}}' ;;
  wait) echo '{"ok":true,"data":{"tasks":[{"status":"success"}]}}' ;;
  *) echo '{"ok":false,"error":{"message":"unexpected","status_code":400,"body":"bad"}}' ;;
esac
`)
	t.Setenv("EXECGO_EXECGOCLI", cli)
	runDir := t.TempDir()
	res := DefaultBus().Call(Context{
		RunID: "run_runtime", TraceID: "trace_runtime", RepoRoot: t.TempDir(), RunDir: runDir,
	}, CallRequest{
		Tool: "runtime.command",
		Args: map[string]any{
			"program":        "/bin/echo",
			"args":           []any{"hello"},
			"sandboxProfile": "container",
			"networkPolicy":  "allowlist",
			"allowedHosts":   []any{"api.openai.com:443"},
			"memoryMB":       512,
			"cpuMillis":      2000,
			"secretRefs": []any{
				map[string]any{"name": "OPENAI_API_KEY", "env": "OPENAI_API_KEY"},
			},
		},
	})
	if !res.OK {
		t.Fatalf("expected ok: %+v", res)
	}
	if res.Output["taskId"] != "task_runtime_1" {
		t.Fatalf("taskId=%v want task_runtime_1", res.Output["taskId"])
	}
	policy, ok := res.Output["policy"].(map[string]any)
	if !ok {
		t.Fatalf("missing policy summary in %+v", res.Output)
	}
	if policy["sandboxProfile"] != "container" || policy["networkPolicy"] != "allowlist" || policy["secretRefCount"] != 1 {
		t.Fatalf("policy=%+v want container allowlist with one secret ref", policy)
	}
	actionFile, _ := res.Output["actionFile"].(string)
	if actionFile == "" {
		t.Fatalf("missing actionFile in %+v", res.Output)
	}
	b, err := os.ReadFile(actionFile)
	if err != nil {
		t.Fatal(err)
	}
	text := string(b)
	for _, want := range []string{
		`"kind": "runtime.command"`,
		`"program": "/bin/echo"`,
		`"profile": "container"`,
		`"policy": "allowlist"`,
		`"allowed_hosts": [`,
		`"api.openai.com:443"`,
		`"memory_mb": 512`,
		`"cpu_millis": 2000`,
		`"secret_refs": [`,
		`"name": "OPENAI_API_KEY"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("action JSON missing %q:\n%s", want, text)
		}
	}
}

func TestRuntimeCommandIsDangerousByDefault(t *testing.T) {
	if risk := DefaultBus().ToolRisk("runtime.command"); risk != RiskDanger {
		t.Fatalf("risk=%q want danger", risk)
	}
}

func TestRuntimeCommandReportsMissingBridge(t *testing.T) {
	t.Setenv("EXECGO_EXECGOCLI", filepath.Join(t.TempDir(), "missing-execgocli"))
	res := DefaultBus().Call(Context{RunDir: t.TempDir()}, CallRequest{
		Tool: "runtime.command",
		Args: map[string]any{"program": "/bin/echo"},
	})
	if res.OK {
		t.Fatalf("expected failure: %+v", res)
	}
	if res.FailureClass != "runtime_bridge_unavailable" {
		t.Fatalf("failureClass=%q want runtime_bridge_unavailable", res.FailureClass)
	}
}

func TestRuntimeCommandRejectsSecretRefsWithoutIsolatedSandbox(t *testing.T) {
	res := DefaultBus().Call(Context{RunDir: t.TempDir()}, CallRequest{
		Tool: "runtime.command",
		Args: map[string]any{
			"program":    "/bin/echo",
			"secretRefs": []any{"OPENAI_API_KEY"},
		},
	})
	if res.OK {
		t.Fatalf("expected policy failure: %+v", res)
	}
	if res.FailureClass != "runtime_policy" {
		t.Fatalf("failureClass=%q want runtime_policy", res.FailureClass)
	}
	if !strings.Contains(res.Error, "isolated sandboxProfile") {
		t.Fatalf("error=%q", res.Error)
	}
}

func TestRuntimeCommandRejectsInvalidNetworkPolicy(t *testing.T) {
	res := DefaultBus().Call(Context{RunDir: t.TempDir()}, CallRequest{
		Tool: "runtime.command",
		Args: map[string]any{
			"program":       "/bin/echo",
			"networkPolicy": "deny",
			"allowedHosts":  []any{"https://example.com"},
		},
	})
	if res.OK {
		t.Fatalf("expected policy failure: %+v", res)
	}
	if res.FailureClass != "runtime_policy" {
		t.Fatalf("failureClass=%q want runtime_policy", res.FailureClass)
	}
}

func TestRuntimeCommandRejectsNetworkPolicyInProcessSandbox(t *testing.T) {
	res := DefaultBus().Call(Context{RunDir: t.TempDir()}, CallRequest{
		Tool: "runtime.command",
		Args: map[string]any{
			"program":       "/bin/echo",
			"networkPolicy": "deny",
		},
	})
	if res.OK {
		t.Fatalf("expected policy failure: %+v", res)
	}
	if res.FailureClass != "runtime_policy" {
		t.Fatalf("failureClass=%q want runtime_policy", res.FailureClass)
	}
	if !strings.Contains(res.Error, "networkPolicy requires an isolated sandboxProfile") {
		t.Fatalf("error=%q", res.Error)
	}
}

func TestRuntimeCommandRejectsEmptyAllowlist(t *testing.T) {
	res := DefaultBus().Call(Context{RunDir: t.TempDir()}, CallRequest{
		Tool: "runtime.command",
		Args: map[string]any{
			"program":        "/bin/echo",
			"sandboxProfile": "container",
			"networkPolicy":  "allowlist",
			"memoryMB":       512,
			"cpuMillis":      1000,
		},
	})
	if res.OK {
		t.Fatalf("expected policy failure: %+v", res)
	}
	if res.FailureClass != "runtime_policy" {
		t.Fatalf("failureClass=%q want runtime_policy", res.FailureClass)
	}
	if !strings.Contains(res.Error, "allowedHosts is required") {
		t.Fatalf("error=%q", res.Error)
	}
}

func TestRuntimeCommandRejectsSecretsWithDefaultNetwork(t *testing.T) {
	res := DefaultBus().Call(Context{RunDir: t.TempDir()}, CallRequest{
		Tool: "runtime.command",
		Args: map[string]any{
			"program":        "/bin/echo",
			"sandboxProfile": "container",
			"memoryMB":       512,
			"cpuMillis":      1000,
			"secretRefs":     []any{"OPENAI_API_KEY"},
		},
	})
	if res.OK {
		t.Fatalf("expected policy failure: %+v", res)
	}
	if res.FailureClass != "runtime_policy" {
		t.Fatalf("failureClass=%q want runtime_policy", res.FailureClass)
	}
	if !strings.Contains(res.Error, "explicit networkPolicy") {
		t.Fatalf("error=%q", res.Error)
	}
}

func TestRuntimeCommandRejectsIsolatedSandboxWithoutResourceLimits(t *testing.T) {
	res := DefaultBus().Call(Context{RunDir: t.TempDir()}, CallRequest{
		Tool: "runtime.command",
		Args: map[string]any{
			"program":        "/bin/echo",
			"sandboxProfile": "container",
		},
	})
	if res.OK {
		t.Fatalf("expected policy failure: %+v", res)
	}
	if res.FailureClass != "runtime_policy" {
		t.Fatalf("failureClass=%q want runtime_policy", res.FailureClass)
	}
	if !strings.Contains(res.Error, "requires memoryMB and cpuMillis") {
		t.Fatalf("error=%q", res.Error)
	}
}

func TestRuntimeCommandRejectsInvalidSandboxProfile(t *testing.T) {
	res := DefaultBus().Call(Context{RunDir: t.TempDir()}, CallRequest{
		Tool: "runtime.command",
		Args: map[string]any{
			"program":        "/bin/echo",
			"sandboxProfile": "host",
		},
	})
	if res.OK {
		t.Fatalf("expected policy failure: %+v", res)
	}
	if res.FailureClass != "runtime_policy" {
		t.Fatalf("failureClass=%q want runtime_policy", res.FailureClass)
	}
	if !strings.Contains(res.Error, "sandboxProfile must be") {
		t.Fatalf("error=%q", res.Error)
	}
}

func TestRuntimeCommandRejectsSecretValueFields(t *testing.T) {
	res := DefaultBus().Call(Context{RunDir: t.TempDir()}, CallRequest{
		Tool: "runtime.command",
		Args: map[string]any{
			"program":        "/bin/echo",
			"sandboxProfile": "container",
			"secretRefs": []any{
				map[string]any{"name": "TOKEN", "value": "should-not-pass"},
			},
		},
	})
	if res.OK {
		t.Fatalf("expected policy failure: %+v", res)
	}
	if res.FailureClass != "runtime_policy" {
		t.Fatalf("failureClass=%q want runtime_policy", res.FailureClass)
	}
	if !strings.Contains(res.Error, "only allow name and env") {
		t.Fatalf("error=%q", res.Error)
	}
}

func writeFakeRuntimeExecGoCLI(t *testing.T, body string) string {
	return testutil.WriteFakeExecGoCLI(t, body)
}
