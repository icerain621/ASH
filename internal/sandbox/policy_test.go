package sandbox_test

import (
	"errors"
	"testing"

	"github.com/ash-repwiki/ash/internal/sandbox"
)

func TestResolveSandboxMode_raisesFloorForDanger(t *testing.T) {
	got := sandbox.ResolveSandboxMode("danger", "off", "")
	if got != sandbox.ModeIsolated {
		t.Fatalf("got %q want isolated", got)
	}
}

func TestResolveSandboxMode_raisesFloorForMedium(t *testing.T) {
	got := sandbox.ResolveSandboxMode("medium", "off", "")
	if got != sandbox.ModeWorkspaceWrite {
		t.Fatalf("got %q want workspace-write", got)
	}
}

func TestAuthorize_dangerOffDenied(t *testing.T) {
	err := sandbox.Authorize("danger", sandbox.ModeOff)
	if err == nil || !errors.Is(err, sandbox.ErrPolicyDenied) {
		t.Fatalf("err=%v want ErrPolicyDenied", err)
	}
}

func TestAuthorize_safeOffOK(t *testing.T) {
	if err := sandbox.Authorize("safe", sandbox.ModeOff); err != nil {
		t.Fatal(err)
	}
}

func TestAuthorize_dangerWorkspaceWriteAllowed(t *testing.T) {
	// Floor elevates to isolated at route time; authorize only blocks explicit off.
	if err := sandbox.Authorize("danger", sandbox.ModeWorkspaceWrite); err != nil {
		t.Fatal(err)
	}
}
