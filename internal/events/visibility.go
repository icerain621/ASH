package events

import "strings"

// Event visibility (v5 thin interaction). Stored on the envelope / run_events column,
// not inside validated payload JSON (payload schemas keep additionalProperties:false).
const (
	VisibilityModelVisible = "model_visible"
	VisibilityUIOnly       = "ui_only"
	VisibilityAudit        = "audit"
)

// DefaultVisibility returns the default visibility for an event type when none is stored.
func DefaultVisibility(eventType string) string {
	t := strings.TrimSpace(eventType)
	switch {
	case strings.HasPrefix(t, "metric."),
		strings.HasPrefix(t, "score."),
		strings.HasPrefix(t, "audit."):
		return VisibilityAudit
	case strings.HasPrefix(t, "ui."),
		t == "gate.waiting_approval":
		return VisibilityUIOnly
	default:
		return VisibilityModelVisible
	}
}

// NormalizeVisibility returns a canonical visibility value.
// Empty/invalid values fall back to DefaultVisibility(eventType).
func NormalizeVisibility(eventType, visibility string) string {
	v := strings.TrimSpace(visibility)
	switch v {
	case VisibilityModelVisible, VisibilityUIOnly, VisibilityAudit:
		return v
	default:
		return DefaultVisibility(eventType)
	}
}

type appendConfig struct {
	visibility string
}

// AppendOption customizes Append behavior.
type AppendOption func(*appendConfig)

// WithVisibility forces an explicit visibility on Append (must be a known value or
// it will be normalized against the event type).
func WithVisibility(visibility string) AppendOption {
	return func(c *appendConfig) {
		c.visibility = visibility
	}
}

func resolveAppendVisibility(eventType string, opts []AppendOption) string {
	cfg := appendConfig{}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	if strings.TrimSpace(cfg.visibility) == "" {
		return DefaultVisibility(eventType)
	}
	return NormalizeVisibility(eventType, cfg.visibility)
}
