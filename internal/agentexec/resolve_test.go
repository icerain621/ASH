package agentexec

import "testing"

func TestResolveAndNormalize(t *testing.T) {
	if AdapterNameOf(Resolve("static")) != "static" {
		t.Fatal("static")
	}
	if AdapterNameOf(Resolve("execgo")) != "execgo_codex" {
		t.Fatal("execgo")
	}
	if AdapterNameOf(Resolve("acp_sdk")) != "acp_sdk" {
		t.Fatal("acp_sdk adapter")
	}
	if NormalizeProviderKind("") != "execgo" {
		t.Fatal("default kind")
	}
	if NormalizeProviderKind("execgo_codex") != "execgo" {
		t.Fatal("alias")
	}
}
