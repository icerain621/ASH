package sandbox_test

import (
	"testing"

	"github.com/ash-repwiki/ash/internal/sandbox"
)

func TestResolveSandboxMode_profileDefault(t *testing.T) {
	got := sandbox.ResolveSandboxMode("safe", "workspace-write", "")
	if got != "workspace-write" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveSandboxMode_overrideCannotLowerBelowRiskFloor(t *testing.T) {
	got := sandbox.ResolveSandboxMode("danger", "isolated", "read-only")
	if got != "isolated" {
		t.Fatalf("got %q want isolated", got)
	}
}

func TestResolveSandboxMode_emptyFallsBackOff(t *testing.T) {
	got := sandbox.ResolveSandboxMode("safe", "", "")
	if got != "off" {
		t.Fatalf("got %q", got)
	}
}

func TestNoopRouterAlwaysInProcess(t *testing.T) {
	r := sandbox.NoopRouter{}
	d, err := r.Route(sandbox.RouteRequest{
		Tool: "bash", Risk: "danger", ProfileDefaultMode: "isolated",
	})
	if err != nil {
		t.Fatal(err)
	}
	if d.Mode != "isolated" {
		t.Fatalf("mode=%q", d.Mode)
	}
	if d.Executor != "in-process" {
		t.Fatalf("executor=%q", d.Executor)
	}
}
