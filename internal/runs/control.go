package runs

import (
	"errors"
	"fmt"

	"github.com/ash-repwiki/ash/internal/rules"
	"github.com/ash-repwiki/ash/internal/store"
	"gorm.io/gorm"
)

var (
	ErrRunNotFound      = errors.New("run not found")
	ErrRunNotResumable  = errors.New("run is not resumable")
	ErrRunMetaMissing   = errors.New("run metadata missing")
	ErrInvalidReplayMode = errors.New("replay mode must be exact or latest_memory")
)

// Replay creates a new run from a completed or failed source run.
func (s *Service) Replay(sourceRunID string, req ReplayRequest) (*ReplayResponse, error) {
	if req.Mode != "exact" && req.Mode != "latest_memory" {
		return nil, ErrInvalidReplayMode
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
	if err := s.db.First(&rec, "id = ?", runID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRunNotFound
		}
		return nil, err
	}
	if rec.Status != "failed" {
		return nil, fmt.Errorf("%w: status is %q", ErrRunNotResumable, rec.Status)
	}

	meta, err := loadRunMeta(s.db.RunDir(runID))
	if err != nil {
		return nil, ErrRunMetaMissing
	}

	rec.Status = "running"
	rec.Recovered = true
	rec.ErrorCode = ""
	rec.ErrorMessage = ""
	rec.FinishedAt = nil
	if err := s.db.Save(&rec).Error; err != nil {
		return nil, err
	}

	if _, err := s.events.Append(runID, rec.TraceID, "run.resumed", "info", map[string]any{
		"fromStatus": "failed",
		"recovered":  true,
	}); err != nil {
		return nil, err
	}

	createReq := CreateRequest{
		Scenario:      meta.Scenario,
		Inputs:        meta.Inputs,
		PolicyProfile: meta.PolicyProfile,
		Repo:          meta.Repo,
	}
	doc, err := s.scenarios.Get(rec.ScenarioName, rec.ScenarioVersion)
	if err != nil {
		return nil, err
	}
	eng := rules.NewEngine(doc)
	if err := s.executeSteps(&rec, createReq, doc, eng, rec.StartedAt); err != nil {
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
