package modelrouter

import "testing"

func TestRouteFallsBackWhenPrimaryUnavailable(t *testing.T) {
	t.Setenv("ASH_MODEL_PRIMARY_STATUS", "unavailable")
	t.Setenv("ASH_MODEL_PRIMARY_MODEL", "primary-model")
	t.Setenv("ASH_MODEL_FALLBACK_STATUS", "available")
	t.Setenv("ASH_MODEL_FALLBACK_MODEL", "fallback-model")
	t.Setenv("ASH_MODEL_FALLBACK_INPUT_MICROS_PER_1K", "2000")
	t.Setenv("ASH_MODEL_FALLBACK_OUTPUT_MICROS_PER_1K", "6000")

	decision := NewFromEnv().Route(Request{Prompt: "route this prompt", OutputTokens: 10})
	if decision.Provider.ID != "fallback" {
		t.Fatalf("provider=%q want fallback", decision.Provider.ID)
	}
	if !decision.FallbackUsed {
		t.Fatal("expected fallbackUsed")
	}
	if decision.Status != "routed" {
		t.Fatalf("status=%q want routed", decision.Status)
	}
	if decision.CostMicros <= 0 {
		t.Fatalf("costMicros=%d want positive", decision.CostMicros)
	}
}

func TestRouteReportsNotConfiguredWithoutProviders(t *testing.T) {
	t.Setenv("ASH_MODEL_PRIMARY_STATUS", "not_configured")
	t.Setenv("ASH_MODEL_FALLBACK_STATUS", "not_configured")

	decision := NewFromEnv().Route(Request{Prompt: "hello"})
	if decision.Status != "not_configured" {
		t.Fatalf("status=%q want not_configured", decision.Status)
	}
	if decision.Provider.ID != "primary" {
		t.Fatalf("provider=%q want primary", decision.Provider.ID)
	}
}
