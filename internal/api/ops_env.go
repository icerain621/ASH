package api

import (
	"os"

	"github.com/ash-repwiki/ash/internal/alerts"
	ashotel "github.com/ash-repwiki/ash/internal/observability/otel"
)

// WorkerOpsSnapshot captures runtime observability flags for /readyz and Scale.
type WorkerOpsSnapshot struct {
	OtelEnabled               bool
	AlertsEvalInterval        string
	MetricsEventReplayEnabled bool
}

func workerOpsSnapshot() WorkerOpsSnapshot {
	out := WorkerOpsSnapshot{
		OtelEnabled:               ashotel.Enabled(),
		MetricsEventReplayEnabled: os.Getenv("ASH_METRICS_EVENT_REPLAY") == "1",
	}
	if d, ok := alerts.ParseEvalInterval(os.Getenv("ASH_ALERTS_EVAL_INTERVAL")); ok {
		out.AlertsEvalInterval = d.String()
	}
	return out
}
