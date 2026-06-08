package otel

import (
	"context"
	"testing"

	obsconfig "github.com/ash-repwiki/ash/internal/observability/config"
)

func TestInit_disabledNoop(t *testing.T) {
	shutdown, err := Init(&obsconfig.OtelPlugin{Enabled: false}, "ash-test")
	if err != nil {
		t.Fatal(err)
	}
	if Enabled() {
		t.Fatal("expected disabled")
	}
	if err := shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestInit_enabledRequiresEndpoint(t *testing.T) {
	_, err := Init(&obsconfig.OtelPlugin{Enabled: true}, "ash-test")
	if err == nil {
		t.Fatal("expected endpoint required error")
	}
}

func TestRuntimeStatus_envOverride(t *testing.T) {
	t.Setenv("ASH_OTEL_ENABLED", "1")
	t.Setenv("ASH_OTEL_ENDPOINT", "localhost:4317")
	st := RuntimeStatus(&obsconfig.OtelPlugin{Enabled: false})
	if st.Endpoint != "localhost:4317" {
		t.Fatalf("endpoint=%q", st.Endpoint)
	}
}
