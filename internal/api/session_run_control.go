package api

import (
	"github.com/ash-repwiki/ash/internal/runs"
	"github.com/ash-repwiki/ash/internal/session"
)

// sessionRunControl adapts runs.Service to session.RunControl without a package cycle.
type sessionRunControl struct {
	runs *runs.Service
}

func (a sessionRunControl) ApproveRun(runID string, req session.GateApproveRequest) error {
	if a.runs == nil {
		return runs.ErrRunNotFound
	}
	_, err := a.runs.Approve(runID, runs.ApproveRequest{
		ActorID: req.ActorID, Reason: req.Reason, Scope: req.Scope, Tool: req.Tool,
	})
	return err
}

func (a sessionRunControl) CancelRun(runID string) error {
	if a.runs == nil {
		return runs.ErrRunNotFound
	}
	_, err := a.runs.Cancel(runID)
	return err
}
