package runs

import (
	"context"
	"strings"
)

// SessionProviderBind mirrors provider selection when linking a run to a session document.
type SessionProviderBind struct {
	Kind     string
	Adapter  string
	Fallback bool
	Reason   string
}

// SessionLinker binds ACP runs to agent session documents without importing internal/session
// (avoids session → goal → runs → session import cycles).
type SessionLinker interface {
	EnsureForRun(spaceID, runID, repoRoot, createdBy string, bind SessionProviderBind) (sessionID string, created bool, err error)
	WithContext(ctx context.Context) SessionLinker
}

// linkProviderSession ensures an agents session for ACP provider selections and emits session.linked.
func (s *Service) linkProviderSession(spaceID, runID, traceID, repoRoot string, sel ProviderSelection) string {
	if s == nil || s.sessionSvc == nil {
		return ""
	}
	if sel.RequestedKind != "acp_sdk" && sel.Adapter != "acp_sdk" {
		return ""
	}
	sessionID, created, err := s.sessionSvc.EnsureForRun(spaceID, runID, repoRoot, "runs", SessionProviderBind{
		Kind: firstNonEmpty(sel.RequestedKind, "acp_sdk"), Adapter: sel.Adapter,
		Fallback: sel.Fallback, Reason: sel.Reason,
	})
	if err != nil || strings.TrimSpace(sessionID) == "" {
		return ""
	}
	_, _ = s.eventsFor().Append(runID, firstNonEmpty(traceID, runID), "session.linked", "info", map[string]any{
		"sessionId":     sessionID,
		"created":       created,
		"requestedKind": sel.RequestedKind,
		"adapter":       sel.Adapter,
		"fallback":      sel.Fallback,
		"reason":        sel.Reason,
	})
	return sessionID
}
