package opsenv

import (
	"testing"
)

func TestLoad(t *testing.T) {
	t.Setenv("ASH_METRICS_EVENT_REPLAY", "1")
	t.Setenv("ASH_ALERTS_EVAL_INTERVAL", "5m")
	t.Setenv("ASH_MEMORY_TTL_SWEEP_INTERVAL", "24h")
	snap := Load()
	if !snap.MetricsEventReplayEnabled {
		t.Fatal("expected metrics replay")
	}
	if snap.AlertsEvalInterval != "5m0s" {
		t.Fatalf("alerts interval=%q want 5m0s", snap.AlertsEvalInterval)
	}
	if snap.MemoryTTLSweepInterval != "24h0m0s" {
		t.Fatalf("ttl sweep interval=%q want 24h0m0s", snap.MemoryTTLSweepInterval)
	}
}
