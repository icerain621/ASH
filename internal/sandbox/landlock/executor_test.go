package landlock_test

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/ash-repwiki/ash/internal/sandbox"
	"github.com/ash-repwiki/ash/internal/sandbox/landlock"
)

func TestAvailableNeverPanics(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Available panicked: %v", r)
		}
	}()
	_ = landlock.Available()
}

func TestAvailableFalseOnNonLinux(t *testing.T) {
	if runtime.GOOS == "linux" {
		t.Skip("linux may report true")
	}
	if landlock.Available() {
		t.Fatal("Available() must be false on non-linux")
	}
}

func TestExecutorUnsupportedOnNonLinux(t *testing.T) {
	if runtime.GOOS == "linux" {
		t.Skip("linux uses real executor")
	}
	_, err := (landlock.Executor{}).Dispatch(context.Background(), sandbox.DispatchRequest{
		Program:  "true",
		RepoRoot: t.TempDir(),
		Timeout:  time.Second,
	})
	if err == nil {
		t.Fatal("expected error on non-linux")
	}
	if got := err.Error(); got != "landlock unsupported on this OS" {
		t.Fatalf("error=%q", got)
	}
}
