package api

import (
	"context"

	"github.com/ash-repwiki/ash/internal/runs"
	"github.com/ash-repwiki/ash/internal/session"
)

// sessionRunLinker adapts session.Service to runs.SessionLinker without a package cycle.
type sessionRunLinker struct {
	svc *session.Service
}

func (l sessionRunLinker) EnsureForRun(spaceID, runID, repoRoot, createdBy string, bind runs.SessionProviderBind) (string, bool, error) {
	view, created, err := l.svc.EnsureForRun(spaceID, runID, repoRoot, createdBy, session.ProviderBinding{
		Kind: bind.Kind, Adapter: bind.Adapter, Fallback: bind.Fallback, Reason: bind.Reason,
	})
	if err != nil || view == nil {
		return "", created, err
	}
	return view.ID, created, nil
}

func (l sessionRunLinker) DisabledToolsForRun(runID string) []string {
	return l.svc.DisabledToolsForRun(runID)
}

func (l sessionRunLinker) AllowedToolsSessionForRun(runID string) []string {
	return l.svc.AllowedToolsSessionForRun(runID)
}

func (l sessionRunLinker) AddAllowedToolSession(runID, tool string) error {
	return l.svc.AddAllowedToolSession(runID, tool)
}

func (l sessionRunLinker) WithContext(ctx context.Context) runs.SessionLinker {
	return sessionRunLinker{svc: l.svc.WithContext(ctx)}
}

// DrainFollowUpForRun starts one queued follow-up after the bound run finishes or fails.
// steer remains the interrupt path; this does not cancel anything.
func (l sessionRunLinker) DrainFollowUpForRun(runID string) {
	if l.svc == nil {
		return
	}
	l.svc.DrainFollowUpForRun(runID)
}
