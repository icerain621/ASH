package runs

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// RunMeta is persisted as run.json for replay/resume (appendix F).
type RunMeta struct {
	RunID           string         `json:"runId"`
	TraceID         string         `json:"traceId"`
	Scenario        ScenarioRef    `json:"scenario"`
	Inputs          map[string]any `json:"inputs"`
	PolicyProfile   string         `json:"policyProfile,omitempty"`
	Repo            *RepoRef       `json:"repo,omitempty"`
	SourceRunID     string         `json:"sourceRunId,omitempty"`
	ReplayMode      string         `json:"replayMode,omitempty"`
}

func saveRunMeta(runDir string, meta RunMeta) error {
	b, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(runDir, "run.json"), append(b, '\n'), 0o644)
}

func loadRunMeta(runDir string) (*RunMeta, error) {
	b, err := os.ReadFile(filepath.Join(runDir, "run.json"))
	if err != nil {
		return nil, err
	}
	var meta RunMeta
	if err := json.Unmarshal(b, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

// ReplayRequest starts a new run from an existing one.
type ReplayRequest struct {
	Mode      string         `json:"mode" binding:"required"` // exact | latest_memory
	Overrides map[string]any `json:"overrides,omitempty"`
}

// ReplayResponse returns the new run id.
type ReplayResponse struct {
	RunID   string `json:"runId"`
	TraceID string `json:"traceId"`
}

// ResumeResponse confirms resume on the same run.
type ResumeResponse struct {
	RunID   string `json:"runId"`
	TraceID string `json:"traceId"`
	Status  string `json:"status"`
}
