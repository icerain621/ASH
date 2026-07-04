package opsenv

import (
	"testing"
)

func TestLoad(t *testing.T) {
	t.Setenv("ASH_METRICS_EVENT_REPLAY", "1")
	t.Setenv("ASH_ALERTS_EVAL_INTERVAL", "5m")
	snap := Load()
	if !snap.MetricsEventReplayEnabled {
		t.Fatal("expected metrics replay")
	}
	if snap.AlertsEvalInterval != "5m0s" {
		t.Fatalf("interval=%q want 5m0s", snap.AlertsEvalInterval)
	}
}
