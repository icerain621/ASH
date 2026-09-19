package sandbox_test

import (
	"errors"
	"testing"

	"github.com/ash-repwiki/ash/internal/execpolicy"
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

func TestAuthorize_dangerWorkspaceWriteDenied(t *testing.T) {
	err := sandbox.Authorize("danger", sandbox.ModeWorkspaceWrite)
	if err == nil || !errors.Is(err, sandbox.ErrPolicyDenied) {
		t.Fatalf("err=%v want ErrPolicyDenied", err)
	}
}

func TestForceIsolatedPolicy(t *testing.T) {
	if !sandbox.ForceIsolatedPolicy("hotfix") || !sandbox.ForceIsolatedPolicy("security") {
		t.Fatal("expected force")
	}
	if sandbox.ForceIsolatedPolicy("default") {
		t.Fatal("default should not force")
	}
}

func TestResolveSandboxModeExt_hotfixForcesIsolated(t *testing.T) {
	got := sandbox.ResolveSandboxModeExt("safe", "workspace-write", "", "", "hotfix", "")
	if got != sandbox.ModeIsolated {
		t.Fatalf("got %q want isolated", got)
	}
}

func TestResolveSandboxModeExt_scenarioMin(t *testing.T) {
	got := sandbox.ResolveSandboxModeExt("safe", "off", "", "isolated", "default", "")
	if got != sandbox.ModeIsolated {
		t.Fatalf("got %q", got)
	}
}

func TestResolveSandboxModeExt_overrideCannotLowerRiskFloor(t *testing.T) {
	got := sandbox.ResolveSandboxModeExt("danger", "isolated", "read-only", "", "", "")
	if got != sandbox.ModeIsolated {
		t.Fatalf("got %q want isolated (override cannot lower)", got)
	}
}

func TestFloorFromExecPolicy_emptyNoFloor(t *testing.T) {
	if got := sandbox.FloorFromExecPolicy(execpolicy.Policy{}); got != "" {
		t.Fatalf("empty policy floor=%q want \"\"", got)
	}
}

func TestFloorFromExecPolicy_fsNoneIsIsolated(t *testing.T) {
	p := execpolicy.Policy{
		Version: execpolicy.SchemaVersion,
		FS:      execpolicy.FSCaps{Mode: execpolicy.FSNone},
	}
	if got := sandbox.FloorFromExecPolicy(p); got != sandbox.ModeIsolated {
		t.Fatalf("floor=%q want isolated", got)
	}
}

func TestFloorFromExecPolicy_networkDenyRaisesIsolated(t *testing.T) {
	p := execpolicy.Policy{
		Version: execpolicy.SchemaVersion,
		Network: execpolicy.NetworkCaps{Egress: execpolicy.NetworkDeny},
	}
	if got := sandbox.FloorFromExecPolicy(p); got != sandbox.ModeIsolated {
		t.Fatalf("floor=%q want isolated", got)
	}
}

func TestResolveSandboxModeExt_emptyExecPolicyDoesNotLower(t *testing.T) {
	got := sandbox.ResolveSandboxModeExt("safe", sandbox.ModeWorkspaceWrite, "", "", "", "")
	if got != sandbox.ModeWorkspaceWrite {
		t.Fatalf("got %q want workspace-write (no execPolicy must not lower)", got)
	}
	// Explicit unrestricted / allow floors must not lower an existing isolated profile.
	loose := sandbox.FloorFromExecPolicy(execpolicy.Policy{
		Version: execpolicy.SchemaVersion,
		FS:      execpolicy.FSCaps{Mode: execpolicy.FSUnrestricted},
		Network: execpolicy.NetworkCaps{Egress: execpolicy.NetworkAllow},
		Process: execpolicy.ProcessCaps{Exec: execpolicy.ProcessAllow},
	})
	got = sandbox.ResolveSandboxModeExt("safe", sandbox.ModeIsolated, "", "", "", loose)
	if got != sandbox.ModeIsolated {
		t.Fatalf("got %q want isolated (loose execPolicy must not lower)", got)
	}
}

func TestResolveSandboxModeExt_stricterExecPolicyRaisesFloor(t *testing.T) {
	floor := sandbox.FloorFromExecPolicy(execpolicy.Policy{
		Version: execpolicy.SchemaVersion,
		FS:      execpolicy.FSCaps{Mode: execpolicy.FSNone},
	})
	got := sandbox.ResolveSandboxModeExt("safe", sandbox.ModeOff, "", "", "", floor)
	if got != sandbox.ModeIsolated {
		t.Fatalf("got %q want isolated (stricter execPolicy raises floor)", got)
	}
}
