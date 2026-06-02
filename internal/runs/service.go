package runs

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ash-repwiki/ash/internal/artifacts"
	"github.com/ash-repwiki/ash/internal/events"
	"github.com/ash-repwiki/ash/internal/rules"
	"github.com/ash-repwiki/ash/internal/store"
	"github.com/ash-repwiki/ash/internal/toolbus"
)

type ScenarioRef struct {
	Name            string `json:"name" binding:"required"`
	ScenarioVersion string `json:"scenarioVersion" binding:"required"`
}

type CreateRequest struct {
	Scenario      ScenarioRef    `json:"scenario" binding:"required"`
	Inputs        map[string]any `json:"inputs" binding:"required"`
	PolicyProfile string         `json:"policyProfile"`
	Repo          *RepoRef       `json:"repo"`
}

type RepoRef struct {
	Root     string `json:"root"`
	Revision string `json:"revision"`
	Branch   string `json:"branch"`
}

type CreateResponse struct {
	RunID   string `json:"runId"`
	TraceID string `json:"traceId"`
}

type Summary struct {
	RunID         string      `json:"runId"`
	TraceID       string      `json:"traceId"`
	Scenario      ScenarioRef `json:"scenario"`
	PolicyProfile string      `json:"policyProfile"`
	Status        string      `json:"status"`
	StartedAt     int64       `json:"startedAt"`
	FinishedAt    *int64      `json:"finishedAt,omitempty"`
	Recovered     bool        `json:"recovered"`
	InputsDigest  string      `json:"inputsDigest,omitempty"`
	Repo          *RepoRef    `json:"repo,omitempty"`
}

type Service struct {
	db        *store.DB
	events    *events.Service
	scenarios *rules.Loader
	tools     *toolbus.Bus
}

func NewService(db *store.DB, ev *events.Service, scenarios *rules.Loader, tools *toolbus.Bus) *Service {
	if tools == nil {
		tools = toolbus.DefaultBus()
	}
	return &Service{db: db, events: ev, scenarios: scenarios, tools: tools}
}

func (s *Service) Create(req CreateRequest) (*CreateResponse, error) {
	return s.createAndExecute(req, createOptions{})
}

func (s *Service) evaluateGate(ctx toolbus.Context, gate rules.Gate) (denied bool, reason string) {
	if gate.Check.Tool == "" {
		return false, ""
	}
	res := s.tools.Call(ctx, toolbus.CallRequest{Tool: gate.Check.Tool})
	if !res.OK {
		msg := res.Error
		if gate.OnFail != nil && gate.OnFail.Message != "" {
			msg = gate.OnFail.Message
		}
		return gate.Blocking, msg
	}
	if gate.Check.Tool == "git.status" {
		expectClean, _ := gate.Check.Expect["clean"].(bool)
		clean, _ := res.Output["clean"].(bool)
		if expectClean && !clean {
			msg := "working tree not clean"
			if gate.OnFail != nil && gate.OnFail.Message != "" {
				msg = gate.OnFail.Message
			}
			return true, msg
		}
	}
	return false, ""
}

func (s *Service) failRun(rec *store.RunRecord, runID, traceID string, started time.Time, code, msg string) (*CreateResponse, error) {
	finished := time.Now().UTC()
	rec.Status = "failed"
	rec.FinishedAt = &finished
	rec.ErrorCode = code
	rec.ErrorMessage = msg
	rec.UpdatedAt = finished
	_ = s.db.Save(rec).Error
	_, _ = s.events.Append(runID, traceID, "run.failed", "error", map[string]any{
		"ok":         false,
		"durationMs": finished.Sub(started).Milliseconds(),
		"error":      map[string]any{"code": code, "message": msg, "recoverable": true},
	})
	return nil, fmt.Errorf("%s: %s", code, msg)
}

func checkpointStrategy(doc *rules.Document) string {
	if doc.Scenario.Checkpoint != nil && doc.Scenario.Checkpoint.Strategy != "" {
		return doc.Scenario.Checkpoint.Strategy
	}
	return "per_step"
}

func (s *Service) Get(runID string) (*Summary, error) {
	var rec store.RunRecord
	if err := s.db.First(&rec, "id = ?", runID).Error; err != nil {
		return nil, err
	}
	return recordToSummary(rec), nil
}

func (s *Service) List(limit int) ([]Summary, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	var rows []store.RunRecord
	if err := s.db.Order("started_at desc").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]Summary, 0, len(rows))
	for _, r := range rows {
		out = append(out, *recordToSummary(r))
	}
	return out, nil
}

func (s *Service) Artifacts(runID string) (*artifacts.Manifest, error) {
	if _, err := s.Get(runID); err != nil {
		return nil, err
	}
	return artifacts.LoadManifest(s.db.RunDir(runID))
}

func (s *Service) DB() *store.DB                   { return s.db }
func (s *Service) Events() *events.Service         { return s.events }

func recordToSummary(rec store.RunRecord) *Summary {
	sum := &Summary{
		RunID: rec.ID, TraceID: rec.TraceID,
		Scenario: ScenarioRef{Name: rec.ScenarioName, ScenarioVersion: rec.ScenarioVersion},
		PolicyProfile: rec.PolicyProfile, Status: rec.Status,
		StartedAt: rec.StartedAt.UnixMilli(), Recovered: rec.Recovered, InputsDigest: rec.InputsDigest,
	}
	if rec.FinishedAt != nil {
		ms := rec.FinishedAt.UnixMilli()
		sum.FinishedAt = &ms
	}
	if rec.RepoRoot != "" {
		sum.Repo = &RepoRef{Root: rec.RepoRoot}
	}
	return sum
}

func digestJSON(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(h[:]), nil
}

func digestString(str string) string {
	h := sha256.Sum256([]byte(str))
	return "sha256:" + hex.EncodeToString(h[:])
}
