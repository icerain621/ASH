package pluginabi

import "testing"

func TestSignAndVerifyHMAC(t *testing.T) {
	t.Setenv("ASH_PLUGIN_SIGNING_KEY", "test-plugin-key")
	t.Setenv("ASH_PLUGIN_SIGNING_REQUIRED", "")
	sig := SignHMAC("test-plugin-key", "obs", "1.0.0", "grpc", CurrentABI, "127.0.0.1:19091")
	if sig == "" {
		t.Fatal("empty signature")
	}
	if err := VerifyRegistrationSignature(sig, "obs", "1.0.0", "grpc", CurrentABI, "127.0.0.1:19091"); err != nil {
		t.Fatal(err)
	}
	if err := VerifyRegistrationSignature("deadbeef", "obs", "1.0.0", "grpc", CurrentABI, "127.0.0.1:19091"); err == nil {
		t.Fatal("expected mismatch")
	}
	if err := VerifyRegistrationSignature("", "obs", "1.0.0", "grpc", CurrentABI, "127.0.0.1:19091"); err == nil {
		t.Fatal("expected missing signature")
	}
}

func TestSigningOptionalWithoutKey(t *testing.T) {
	t.Setenv("ASH_PLUGIN_SIGNING_KEY", "")
	t.Setenv("ASH_PLUGIN_SIGNING_REQUIRED", "")
	if err := VerifyRegistrationSignature("", "obs", "1.0.0", "grpc", CurrentABI, "x"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ASH_PLUGIN_SIGNING_REQUIRED", "1")
	if err := VerifyRegistrationSignature("", "obs", "1.0.0", "grpc", CurrentABI, "x"); err == nil {
		t.Fatal("expected misconfiguration error")
	}
}

func TestSignatureFromCapabilities(t *testing.T) {
	got := SignatureFromCapabilities([]string{"export", CapabilitySignPrefix + "abc123"})
	if got != "abc123" {
		t.Fatalf("got %q", got)
	}
}
