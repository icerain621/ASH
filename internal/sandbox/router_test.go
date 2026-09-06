package sandbox_test

import (
	"testing"

	"github.com/ash-repwiki/ash/internal/sandbox"
)

func TestDefaultRouterLandlockWhenOK(t *testing.T) {
	t.Setenv("ASH_SANDBOX_LANDLOCK", "") // default-on
	t.Setenv("ASH_SKIP_SANDBOX", "1")

	r := sandbox.DefaultRouter{
		PreferDocker: false,
		DockerOK:     func() bool { return false },
		LandlockOK:   func() bool { return false },
	}
	dec, err := r.Route(sandbox.RouteRequest{Risk: "danger", ProfileDefaultMode: sandbox.ModeIsolated})
	if err != nil {
		t.Fatal(err)
	}
	if dec.Executor == "landlock" {
		t.Fatalf("executor=%q want fallthrough when LandlockOK=false", dec.Executor)
	}
	if dec.Executor != "process" {
		t.Fatalf("executor=%q want process", dec.Executor)
	}

	r.LandlockOK = func() bool { return true }
	dec, err = r.Route(sandbox.RouteRequest{Risk: "danger", ProfileDefaultMode: sandbox.ModeIsolated})
	if err != nil {
		t.Fatal(err)
	}
	if dec.Executor != "landlock" {
		t.Fatalf("executor=%q want landlock", dec.Executor)
	}
	if dec.Reason != "landlock-available" {
		t.Fatalf("reason=%q", dec.Reason)
	}
}

func TestDefaultRouterLandlockOnlyIsolated(t *testing.T) {
	t.Setenv("ASH_SANDBOX_LANDLOCK", "")
	r := sandbox.DefaultRouter{
		PreferDocker: false,
		DockerOK:     func() bool { return false },
		LandlockOK:   func() bool { return true },
	}
	dec, err := r.Route(sandbox.RouteRequest{Risk: "caution", ProfileDefaultMode: sandbox.ModeWorkspaceWrite})
	if err != nil {
		t.Fatal(err)
	}
	if dec.Executor == "landlock" {
		t.Fatal("landlock must not be selected for workspace-write")
	}
}

func TestDefaultRouterLandlockOptOut(t *testing.T) {
	t.Setenv("ASH_SANDBOX_LANDLOCK", "0")
	r := sandbox.DefaultRouter{
		PreferDocker: false,
		DockerOK:     func() bool { return false },
		LandlockOK:   func() bool { return true },
	}
	dec, err := r.Route(sandbox.RouteRequest{Risk: "danger", ProfileDefaultMode: sandbox.ModeIsolated})
	if err != nil {
		t.Fatal(err)
	}
	if dec.Executor == "landlock" {
		t.Fatal("landlock must not be selected when ASH_SANDBOX_LANDLOCK=0")
	}
	if sandbox.LandlockPreferred() {
		t.Fatal("LandlockPreferred should be false when env=0")
	}
}

func TestLandlockPreferredDefault(t *testing.T) {
	t.Setenv("ASH_SANDBOX_LANDLOCK", "")
	if !sandbox.LandlockPreferred() {
		t.Fatal("default LandlockPreferred want true")
	}
	t.Setenv("ASH_SANDBOX_LANDLOCK", "1")
	if !sandbox.LandlockPreferred() {
		t.Fatal("ASH_SANDBOX_LANDLOCK=1 want preferred")
	}
}
