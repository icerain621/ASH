package execpolicy

import (
	"strings"
	"testing"
)

func TestParse_BadVersion(t *testing.T) {
	_, err := ParsePolicyJSON(`{"version":"execpolicy.v0","network":{"egress":"deny"}}`)
	if err == nil {
		t.Fatal("expected error for bad version")
	}
	if !strings.Contains(err.Error(), SchemaVersion) {
		t.Fatalf("error=%v want mention of schema version", err)
	}
}

func TestParse_InvalidEgress(t *testing.T) {
	_, err := ParsePolicyJSON(`{"version":"ash.execpolicy.v1","network":{"egress":"open"}}`)
	if err == nil {
		t.Fatal("expected error for invalid egress")
	}
}

func TestParse_OK(t *testing.T) {
	p, err := ParsePolicyJSON(`{
		"version":"ash.execpolicy.v1",
		"network":{"egress":"Deny"},
		"fs":{"mode":"read-only"},
		"process":{"exec":"ask"}
	}`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if p.Network.Egress != NetworkDeny || p.FS.Mode != FSReadOnly || p.Process.Exec != ProcessAsk {
		t.Fatalf("policy=%+v", p)
	}
}

func TestFromSpaceBodyJSON_Missing(t *testing.T) {
	p, err := FromSpaceBodyJSON(`{"hooks":{"version":"ash.hooks.v1","rules":[]}}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !p.Empty() {
		t.Fatalf("want empty, got %+v", p)
	}
}

func TestFromSpaceBodyJSON_Present(t *testing.T) {
	p, err := FromSpaceBodyJSON(`{
		"execPolicy":{
			"version":"ash.execpolicy.v1",
			"network":{"egress":"deny"},
			"fs":{"mode":"none"},
			"process":{"exec":"deny"}
		}
	}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Network.Egress != NetworkDeny || p.FS.Mode != FSNone || p.Process.Exec != ProcessDeny {
		t.Fatalf("policy=%+v", p)
	}
}

func TestFromHarnessSpecJSON_SynthesizeSandbox(t *testing.T) {
	p, err := FromHarnessSpecJSON(`{
		"apiVersion":"ash.harness/v1",
		"kind":"HarnessProfileSpec",
		"spec":{
			"provider":{"kind":"static"},
			"sandbox":{"defaultMode":"workspace-write","network":"deny"},
			"tools":{},
			"policyProfile":"default"
		}
	}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Network.Egress != NetworkDeny {
		t.Fatalf("network=%q", p.Network.Egress)
	}
	if p.FS.Mode != FSWorkspaceWrite {
		t.Fatalf("fs=%q", p.FS.Mode)
	}
	if p.Process.Exec != ProcessAllow {
		t.Fatalf("process=%q", p.Process.Exec)
	}
}

func TestFromHarnessSpecJSON_ExplicitExecPolicy(t *testing.T) {
	p, err := FromHarnessSpecJSON(`{
		"execPolicy":{
			"version":"ash.execpolicy.v1",
			"network":{"egress":"ask"},
			"fs":{"mode":"none"},
			"process":{"exec":"deny"}
		},
		"apiVersion":"ash.harness/v1",
		"kind":"HarnessProfileSpec",
		"spec":{
			"sandbox":{"defaultMode":"off","network":"allow"}
		}
	}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Explicit execPolicy wins over sandbox synthesize.
	if p.Network.Egress != NetworkAsk || p.FS.Mode != FSNone || p.Process.Exec != ProcessDeny {
		t.Fatalf("policy=%+v", p)
	}
}
