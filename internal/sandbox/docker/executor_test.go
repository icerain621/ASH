package docker

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ash-repwiki/ash/internal/sandbox"
)

func TestImageDefaultAndOverrides(t *testing.T) {
	e := Executor{}
	if got := e.image(); got != defaultImage {
		t.Fatalf("default image=%q", got)
	}
	e.Image = "custom:tag"
	if got := e.image(); got != "custom:tag" {
		t.Fatalf("explicit image=%q", got)
	}
	e.Image = ""
	t.Setenv("ASH_SANDBOX_IMAGE", "env-image:dev")
	if got := e.image(); got != "env-image:dev" {
		t.Fatalf("env image=%q", got)
	}
}

func TestDispatchSkippedWhenSandboxDisabled(t *testing.T) {
	t.Setenv("ASH_SKIP_SANDBOX", "1")
	res, err := Executor{}.Dispatch(context.Background(), sandbox.DispatchRequest{
		RepoRoot: t.TempDir(),
		Program:  "true",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.OK {
		t.Fatal("want not ok when sandbox skipped")
	}
	if !strings.Contains(res.Error, "docker unavailable") && !strings.Contains(res.Error, "ASH_SKIP_SANDBOX") {
		t.Fatalf("error=%q", res.Error)
	}
}

func TestDispatchValidation(t *testing.T) {
	prevAvail := dockerAvailable
	dockerAvailable = func() bool { return true }
	t.Cleanup(func() { dockerAvailable = prevAvail })

	res, err := Executor{}.Dispatch(context.Background(), sandbox.DispatchRequest{Program: "true"})
	if err != nil {
		t.Fatal(err)
	}
	if res.OK || !strings.Contains(res.Error, "repoRoot") {
		t.Fatalf("want repoRoot error, got %+v", res)
	}

	res, err = Executor{}.Dispatch(context.Background(), sandbox.DispatchRequest{RepoRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if res.OK || !strings.Contains(res.Error, "program") {
		t.Fatalf("want program error, got %+v", res)
	}
}

func TestDispatchSuccessReadOnlyAndArgs(t *testing.T) {
	prevAvail, prevCmd := dockerAvailable, commandContext
	dockerAvailable = func() bool { return true }
	var sawArgs []string
	commandContext = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
		if name != "docker" {
			t.Fatalf("name=%q", name)
		}
		sawArgs = append([]string{}, arg...)
		return exec.CommandContext(ctx, "bash", "-c", "echo hello-from-fake; exit 0")
	}
	t.Cleanup(func() {
		dockerAvailable = prevAvail
		commandContext = prevCmd
	})

	root := t.TempDir()
	res, err := Executor{Image: "img:test"}.Dispatch(nil, sandbox.DispatchRequest{
		RepoRoot:    root,
		Program:     "sh",
		Args:        []string{"-c", "true"},
		SandboxMode: sandbox.ModeReadOnly,
		RunID:       "run1",
		StepID:      "step1",
		Timeout:     time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK || !strings.Contains(res.Stdout, "hello-from-fake") {
		t.Fatalf("res=%+v", res)
	}
	joined := strings.Join(sawArgs, " ")
	if !strings.Contains(joined, ":/workspace:ro") {
		t.Fatalf("want ro mount in args=%v", sawArgs)
	}
	if !strings.Contains(joined, "img:test") || !strings.Contains(joined, "ASH_RUN_ID=run1") {
		t.Fatalf("args=%v", sawArgs)
	}
	abs, _ := filepath.Abs(root)
	if !strings.Contains(joined, abs) {
		t.Fatalf("want abs root in args=%v", sawArgs)
	}
}

func TestDispatchImageMissingAndExitError(t *testing.T) {
	prevAvail, prevCmd := dockerAvailable, commandContext
	dockerAvailable = func() bool { return true }
	t.Cleanup(func() {
		dockerAvailable = prevAvail
		commandContext = prevCmd
	})

	commandContext = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "bash", "-c", "echo Unable to find image xyz >&2; exit 1")
	}
	res, err := Executor{Image: "missing:tag"}.Dispatch(context.Background(), sandbox.DispatchRequest{
		RepoRoot: t.TempDir(),
		Program:  "true",
		Timeout:  time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.OK || !strings.Contains(res.Error, "image missing:tag missing") {
		t.Fatalf("want image missing wrap, got %+v", res)
	}
	if res.ExitCode == 0 {
		t.Fatalf("exit=%d", res.ExitCode)
	}

	commandContext = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "bash", "-c", "echo boom >&2; exit 3")
	}
	res, err = Executor{}.Dispatch(context.Background(), sandbox.DispatchRequest{
		RepoRoot: t.TempDir(),
		Program:  "true",
		Timeout:  time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.OK || res.ExitCode != 3 {
		t.Fatalf("want exit 3, got %+v", res)
	}
}

func TestDispatchNonExitError(t *testing.T) {
	prevAvail, prevCmd := dockerAvailable, commandContext
	dockerAvailable = func() bool { return true }
	commandContext = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "bash", "-c", "kill -9 $$")
	}
	t.Cleanup(func() {
		dockerAvailable = prevAvail
		commandContext = prevCmd
	})
	res, err := Executor{}.Dispatch(context.Background(), sandbox.DispatchRequest{
		RepoRoot: t.TempDir(),
		Program:  "true",
		Timeout:  time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.OK {
		t.Fatalf("want failure %+v", res)
	}
	if res.ExitCode == 0 {
		// signal death may still surface as ExitError with non-zero code
		t.Logf("non-zero preferred; got exit=%d err=%q", res.ExitCode, res.Error)
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate("  ok  "); got != "ok" {
		t.Fatalf("got %q", got)
	}
	long := strings.Repeat("a", 5000)
	got := truncate(long)
	if !strings.HasSuffix(got, "...<truncated>") {
		t.Fatalf("suffix missing: len=%d", len(got))
	}
	if len(got) >= len(long) {
		t.Fatalf("expected truncation len=%d", len(got))
	}
}
