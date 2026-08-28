package process_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/ash-repwiki/ash/internal/sandbox"
	"github.com/ash-repwiki/ash/internal/sandbox/process"
)

func TestProcessDispatchEcho(t *testing.T) {
	root := t.TempDir()
	program, args := echoArgs("hello-sandbox")
	res, err := (process.Executor{}).Dispatch(context.Background(), sandbox.DispatchRequest{
		Program: program, Args: args, RepoRoot: root, Timeout: 5 * time.Second,
		SandboxMode: sandbox.ModeWorkspaceWrite, RunID: "run_x", StepID: "s1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK {
		t.Fatalf("res=%+v", res)
	}
	if res.Stdout == "" && runtime.GOOS != "windows" {
		t.Fatalf("empty stdout %+v", res)
	}
}

func TestProcessRejectsOutsidePathArg(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(filepath.Dir(root), "outside-"+filepath.Base(root))
	_ = os.WriteFile(outside, []byte("x"), 0o644)
	program, _ := echoArgs("x")
	res, err := (process.Executor{}).Dispatch(context.Background(), sandbox.DispatchRequest{
		Program: program, Args: []string{outside}, RepoRoot: root, Timeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.OK {
		t.Fatalf("expected deny, got %+v", res)
	}
}

func echoArgs(msg string) (string, []string) {
	if runtime.GOOS == "windows" {
		return "cmd", []string{"/C", "echo", msg}
	}
	return "echo", []string{msg}
}
