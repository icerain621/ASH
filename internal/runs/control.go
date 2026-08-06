package runs

import (
	"context"
	"errors"
	"fmt"

	"github.com/ash-repwiki/ash/internal/rules"
	"github.com/ash-repwiki/ash/internal/store"
	"gorm.io/gorm"
)

var (
	ErrRunNotFound       = errors.New("run not found")
	ErrRunNotResumable   = errors.New("run is not resumable")
	ErrRunNotApprovable  = errors.New("run is not approvable")
	ErrRunNotReplayable  = errors.New("run is not replayable")
	ErrRunMetaMissing    = errors.New("run metadata missing")
	ErrInvalidReplayMode = errors.New("replay mode must be exact or latest_memory")
)

// canReplay reports whether a source run may be replayed into a new run.
func canReplay(status string) bool {
	switch status {
	case StatusFinished, StatusFailed, StatusCanceled:
		return true
	default:
		return false
	}
}

// Replay creates a new run from a completed, failed, or canceled source run.
func (s *Service) Replay(sourceRunID string, req ReplayRequest) (*ReplayResponse, error) {
	if req.Mode != "exact" && req.Mode != "latest_memory" {
		return nil, ErrInvalidReplayMode
	}
	var sourceRec store.RunRecord
	if err := s.gdb().First(&sourceRec, "id = ?", sourceRunID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRunNotFound
		}
		return nil, err
	}
	if !canReplay(sourceRec.Status) {
		return nil, fmt.Errorf("%w: status is %q", ErrRunNotReplayable, sourceRec.Status)
	}
	meta, err := s.loadMetaForRun(sourceRunID)
	if err != nil {
		return nil, err
	}

	inputs := copyMap(meta.Inputs)
	for k, v := range req.Overrides {
		inputs[k] = v
	}
	if req.Mode == "latest_memory" {
		inputs["useLatestMemory"] = true
	}

	createReq := CreateRequest{
		Scenario:      meta.Scenario,
		Inputs:        inputs,
		PolicyProfile: meta.PolicyProfile,
		ActorRole:     firstNonEmpty(meta.ActorRole, sourceRec.ActorRole),
		SpaceID:       sourceRec.SpaceID,
		Repo:          meta.Repo,
	}
	resp, err := s.CreateWithOptions(createReq, createOptions{
		sourceRunID: sourceRunID,
		replayMode:  req.Mode,
	})
	if err != nil {
		return nil, err
	}
	return &ReplayResponse{RunID: resp.RunID, TraceID: resp.TraceID}, nil
}

// Resume re-executes a failed run on the same run id (M0: full re-run, append events).
func (s *Service) Resume(runID string) (*ResumeResponse, error) {
	var rec store.RunRecord
	if err := s.gdb().First(&rec, "id = ?", runID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRunNotFound
		}
		return nil, err
	}
	if !canResume(rec.Status) {
		return nil, fmt.Errorf("%w: status is %q", ErrRunNotResumable, rec.Status)
	}

	meta, err := loadRunMeta(s.db.RunDir(runID))
	if err != nil {
		return nil, ErrRunMetaMissing
	}

	if err := s.refreshRunStatus(&rec); err != nil {
		return nil, err
	}
	if !canResume(rec.Status) {
		return nil, fmt.Errorf("%w: status is %q", ErrRunNotResumable, rec.Status)
	}
	if err := applyRunStatus(&rec, StatusRunning); err != nil {
		return nil, fmt.Errorf("%w: status is %q", ErrRunNotResumable, rec.Status)
	}
	rec.Recovered = true
	rec.ErrorCode = ""
	rec.ErrorMessage = ""
	rec.FinishedAt = nil
	if err := s.gdb().Save(&rec).Error; err != nil {
		return nil, err
	}

	if _, err := s.eventsFor().Append(runID, rec.TraceID, "run.resumed", "info", map[string]any{
		"fromStatus": StatusFailed,
		"recovered":  true,
	}); err != nil {
		return nil, err
	}

	createReq := CreateRequest{
		Scenario:      meta.Scenario,
		Inputs:        meta.Inputs,
		PolicyProfile: meta.PolicyProfile,
		ActorRole:     firstNonEmpty(meta.ActorRole, rec.ActorRole),
		Repo:          meta.Repo,
	}
	doc, err := s.scenarios.Get(rec.ScenarioName, rec.ScenarioVersion)
	if err != nil {
		return nil, err
	}
	eng := rules.NewEngine(doc)
	if err := s.executeSteps(context.Background(), &rec, createReq, doc, eng, rec.StartedAt); err != nil {
		if errors.Is(err, ErrWaitingApproval) || errors.Is(err, ErrRunCanceled) {
			sum, getErr := s.Get(runID)
			status := rec.Status
			if getErr == nil {
				status = sum.Status
			}
			return &ResumeResponse{RunID: runID, TraceID: rec.TraceID, Status: status}, nil
		}
		return nil, err
	}
	return &ResumeResponse{RunID: runID, TraceID: rec.TraceID, Status: rec.Status}, nil
}

func (s *Service) loadMetaForRun(runID string) (*RunMeta, error) {
	if _, err := s.Get(runID); err != nil {
		return nil, ErrRunNotFound
	}
	meta, err := loadRunMeta(s.db.RunDir(runID))
	if err != nil {
		return nil, ErrRunMetaMissing
	}
	return meta, nil
}

func copyMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

type createOptions struct {
	sourceRunID string
	replayMode  string
}
