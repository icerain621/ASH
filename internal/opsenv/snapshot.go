package opsenv

import (
	"os"

	"github.com/ash-repwiki/ash/internal/alerts"
	"github.com/ash-repwiki/ash/internal/memory"
	ashotel "github.com/ash-repwiki/ash/internal/observability/otel"
)

// Snapshot captures runtime observability flags for /readyz and Scale.
type Snapshot struct {
	OtelEnabled               bool
	AlertsEvalInterval        string
	MemoryTTLSweepInterval    string
	MetricsEventReplayEnabled bool
}

// Load reads worker ops env (OTel, alerts interval, TTL sweep, metrics replay).
func Load() Snapshot {
	out := Snapshot{
		OtelEnabled:               ashotel.Enabled(),
		MetricsEventReplayEnabled: os.Getenv("ASH_METRICS_EVENT_REPLAY") == "1",
	}
	if d, ok := alerts.ParseEvalInterval(os.Getenv("ASH_ALERTS_EVAL_INTERVAL")); ok {
		out.AlertsEvalInterval = d.String()
	}
	if d, ok := memory.ParseSweepInterval(os.Getenv("ASH_MEMORY_TTL_SWEEP_INTERVAL")); ok {
		out.MemoryTTLSweepInterval = d.String()
	}
	return out
}
