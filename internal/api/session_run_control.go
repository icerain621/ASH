package api

import (
	"github.com/ash-repwiki/ash/internal/runs"
)

// sessionRunControl adapts runs.Service to session.RunControl without a package cycle.
type sessionRunControl struct {
	runs *runs.Service
}

func (a sessionRunControl) ApproveRun(runID, actorID, reason string) error {
	if a.runs == nil {
		return runs.ErrRunNotFound
	}
	_, err := a.runs.Approve(runID, runs.ApproveRequest{ActorID: actorID, Reason: reason})
	return err
}

func (a sessionRunControl) CancelRun(runID string) error {
	if a.runs == nil {
		return runs.ErrRunNotFound
	}
	_, err := a.runs.Cancel(runID)
	return err
}
