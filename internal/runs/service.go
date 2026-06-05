package runs

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/ash-repwiki/ash/internal/agentexec"
	"github.com/ash-repwiki/ash/internal/artifacts"
	"github.com/ash-repwiki/ash/internal/artifactstore"
	"github.com/ash-repwiki/ash/internal/config"
	"github.com/ash-repwiki/ash/internal/events"
	"github.com/ash-repwiki/ash/internal/rag"
	"github.com/ash-repwiki/ash/internal/rules"
	"github.com/ash-repwiki/ash/internal/store"
	"github.com/ash-repwiki/ash/internal/toolbus"
)

var ErrWaitingApproval = fmt.Errorf("run waiting for approval")

type ScenarioRef struct {
	Name            string `json:"name" binding:"required"`
	ScenarioVersion string `json:"scenarioVersion" binding:"required"`
}

type CreateRequest struct {
	Scenario      ScenarioRef    `json:"scenario" binding:"required"`
	Inputs        map[string]any `json:"inputs" binding:"required"`
	PolicyProfile string         `json:"policyProfile"`
	Repo          *RepoRef       `json:"repo"`
	SpaceID       string         `json:"spaceId,omitempty"`
	ActorRole     string         `json:"actorRole,omitempty"`
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

type ArtifactAccessResponse struct {
	RunID       string `json:"runId"`
	Name        string `json:"name"`
	URI         string `json:"uri"`
	SignedURL   string `json:"signedUrl"`
	ExpiresAt   int64  `json:"expiresAt"`
	Digest      string `json:"digest"`
	ContentType string `json:"contentType"`
	SizeBytes   int64  `json:"sizeBytes"`
}

type CheckpointAccessResponse struct {
	RunID          string `json:"runId"`
	CheckpointID   string `json:"checkpointId"`
	StepID         string `json:"stepId"`
	URI            string `json:"uri"`
	SignedURL      string `json:"signedUrl"`
	ExpiresAt      int64  `json:"expiresAt"`
	SnapshotDigest string `json:"snapshotDigest"`
	ContentType    string `json:"contentType"`
	SizeBytes      int64  `json:"sizeBytes"`
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
	SpaceID       string      `json:"spaceId,omitempty"`
	ActorRole     string      `json:"actorRole,omitempty"`
}

type TimelineResponse struct {
	Items []TimelineItem `json:"items"`
}

type TimelineItem struct {
	Seq      int64           `json:"seq,omitempty"`
	TS       int64           `json:"ts,omitempty"`
	Type     string          `json:"type"`
	Severity string          `json:"severity,omitempty"`
	StepID   string          `json:"stepId,omitempty"`
	Status   string          `json:"status,omitempty"`
	Payload  json.RawMessage `json:"payload,omitempty"`
}

type CancelResponse struct {
	RunID  string `json:"runId"`
	Status string `json:"status"`
}

type ApproveRequest struct {
	ActorID string `json:"actorId,omitempty"`
	Reason  string `json:"reason,omitempty"`
}

type ApproveResponse struct {
	RunID string `json:"runId"`
	OK    bool   `json:"ok"`
}

type Service struct {
	db        *store.DB
	events    *events.Service
	scenarios *rules.Loader
	tools     *toolbus.Bus
	agent     agentexec.Executor
	rag       *rag.Service
	artifacts artifactstore.Store
}

func NewService(db *store.DB, ev *events.Service, scenarios *rules.Loader, tools *toolbus.Bus) *Service {
	if tools == nil {
		tools = toolbus.DefaultBus()
	}
	cfg := config.Load()
	return &Service{
		db: db, events: ev, scenarios: scenarios, tools: tools,
		agent:     agentexec.NewExecGoCodexExecutor(),
		rag:       rag.NewService(db),
		artifacts: artifactstore.New(cfg.ArtifactStore, db.DataDir()),
	}
}

func (s *Service) WithAgentExecutor(exec agentexec.Executor) *Service {
	if exec != nil {
		s.agent = exec
	}
	return s
}

func (s *Service) AgentAdapter() string {
	if named, ok := s.agent.(interface{ AdapterName() string }); ok {
		return named.AdapterName()
	}
	return "unknown"
}

func (s *Service) WithArtifactStore(store artifactstore.Store) *Service {
	if store != nil {
		s.artifacts = store
	}
	return s
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
	return s.ListForSpace("", limit)
}

func (s *Service) ListForSpace(spaceID string, limit int) ([]Summary, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	var rows []store.RunRecord
	q := s.db.Order("started_at desc").Limit(limit)
	if spaceID != "" {
		q = q.Where("space_id = ?", spaceID)
	}
	if err := q.Find(&rows).Error; err != nil {
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

func (s *Service) Checkpoints(runID string) ([]store.Checkpoint, error) {
	if _, err := s.Get(runID); err != nil {
		return nil, err
	}
	var rows []store.Checkpoint
	if err := s.db.Where("run_id = ?", runID).Order("created_at asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *Service) ArtifactAccess(runID, name string, ttl time.Duration) (*ArtifactAccessResponse, error) {
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}
	if _, err := s.Get(runID); err != nil {
		return nil, err
	}
	var row store.ArtifactIndex
	if err := s.db.Where("run_id = ? AND name = ?", runID, name).First(&row).Error; err != nil {
		return nil, fmt.Errorf("artifact not found: %w", err)
	}
	expires := time.Now().UTC().Add(ttl)
	signed := row.URI
	if s.artifacts != nil {
		key := row.StoreKey
		if key == "" {
			key = filepath.ToSlash(filepath.Join("runs", runID, row.Name))
		}
		url, err := s.artifacts.SignedURL(context.Background(), key, ttl)
		if err != nil {
			return nil, err
		}
		signed = url
	}
	_ = s.writeAudit(runID, "", "artifact.access_url_issued", map[string]any{
		"name": name, "digest": row.Digest, "expiresAt": expires.UnixMilli(),
	})
	return &ArtifactAccessResponse{
		RunID: runID, Name: row.Name, URI: row.URI, SignedURL: signed, ExpiresAt: expires.UnixMilli(),
		Digest: row.Digest, ContentType: row.ContentType, SizeBytes: row.SizeBytes,
	}, nil
}

func (s *Service) CheckpointAccess(runID, checkpointID string, ttl time.Duration) (*CheckpointAccessResponse, error) {
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}
	if _, err := s.Get(runID); err != nil {
		return nil, err
	}
	var row store.Checkpoint
	if err := s.db.Where("run_id = ? AND id = ?", runID, checkpointID).First(&row).Error; err != nil {
		return nil, fmt.Errorf("checkpoint not found: %w", err)
	}
	expires := time.Now().UTC().Add(ttl)
	signed := row.URI
	if s.artifacts != nil {
		key := row.StoreKey
		if key == "" {
			key = filepath.ToSlash(filepath.Join("runs", runID, "checkpoints", row.ID+".json"))
		}
		url, err := s.artifacts.SignedURL(context.Background(), key, ttl)
		if err != nil {
			return nil, err
		}
		signed = url
	}
	_ = s.writeAudit(runID, "", "checkpoint.access_url_issued", map[string]any{
		"checkpointId": checkpointID, "stepId": row.StepID,
		"snapshotDigest": row.SnapshotDigest, "expiresAt": expires.UnixMilli(),
	})
	return &CheckpointAccessResponse{
		RunID: runID, CheckpointID: row.ID, StepID: row.StepID, URI: row.URI,
		SignedURL: signed, ExpiresAt: expires.UnixMilli(), SnapshotDigest: row.SnapshotDigest,
		ContentType: row.ContentType, SizeBytes: row.SizeBytes,
	}, nil
}

func (s *Service) Timeline(runID string, limit int) (*TimelineResponse, error) {
	if limit <= 0 || limit > 1000 {
		limit = 500
	}
	if _, err := s.Get(runID); err != nil {
		return nil, err
	}
	evs, err := s.events.ListAfter(runID, 0, limit)
	if err != nil {
		return nil, err
	}
	out := make([]TimelineItem, 0, len(evs))
	for _, ev := range evs {
		out = append(out, TimelineItem{
			Seq: ev.Seq, TS: ev.TS, Type: ev.Type, Severity: ev.Severity, Payload: ev.Payload,
		})
	}
	return &TimelineResponse{Items: out}, nil
}

func (s *Service) ToolCalls(runID string) ([]store.ToolCall, error) {
	var rows []store.ToolCall
	if err := s.db.Where("run_id = ?", runID).Order("created_at asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *Service) AgentTasks(runID string) ([]store.AgentTask, error) {
	var rows []store.AgentTask
	if err := s.db.Where("run_id = ?", runID).Order("created_at asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *Service) QualityMetrics(runID string) ([]store.QualityMetric, error) {
	if _, err := s.Get(runID); err != nil {
		return nil, err
	}
	var rows []store.QualityMetric
	if err := s.db.Where("run_id = ?", runID).Order("created_at asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *Service) Cancel(runID string) (*CancelResponse, error) {
	var rec store.RunRecord
	if err := s.db.First(&rec, "id = ?", runID).Error; err != nil {
		return nil, ErrRunNotFound
	}
	if rec.Status == "finished" || rec.Status == "failed" || rec.Status == "canceled" {
		return &CancelResponse{RunID: runID, Status: rec.Status}, nil
	}
	var tasks []store.AgentTask
	_ = s.db.Where("run_id = ? AND status IN ?", runID, []string{"running", "accepted"}).Find(&tasks).Error
	for _, task := range tasks {
		_ = s.agent.Cancel(context.Background(), firstNonEmpty(task.ExecGoTaskID, task.ActionID, task.ID))
	}
	now := time.Now().UTC()
	rec.Status = "canceled"
	rec.FinishedAt = &now
	rec.UpdatedAt = now
	if err := s.db.Save(&rec).Error; err != nil {
		return nil, err
	}
	_, _ = s.events.Append(runID, rec.TraceID, "run.canceled", "warn", map[string]any{"status": "canceled"})
	s.decidePendingApproval(runID, "", "canceled", "", "run canceled")
	return &CancelResponse{RunID: runID, Status: rec.Status}, nil
}

func (s *Service) Approve(runID string, req ApproveRequest) (*ApproveResponse, error) {
	var rec store.RunRecord
	if err := s.db.First(&rec, "id = ?", runID).Error; err != nil {
		return nil, ErrRunNotFound
	}
	if rec.Status != "waiting_approval" {
		return nil, fmt.Errorf("%w: status is %q", ErrRunNotResumable, rec.Status)
	}

	var step store.RunStep
	if err := s.db.Where("run_id = ? AND status = ?", runID, "waiting_approval").
		Order("updated_at desc").
		First(&step).Error; err != nil {
		return nil, fmt.Errorf("waiting approval step not found: %w", err)
	}

	meta, err := loadRunMeta(s.db.RunDir(runID))
	if err != nil {
		return nil, ErrRunMetaMissing
	}
	if meta.Inputs == nil {
		meta.Inputs = map[string]any{}
	}
	approvalKind := "human"
	if step.ErrorCode == "GATE_CITATION_MISSING" {
		approvalKind = "citation"
		appendApprovedStep(meta.Inputs, "_approvedCitationSteps", step.StepID)
	} else if step.ErrorCode == "TOOL_DANGEROUS_APPROVAL_REQUIRED" {
		approvalKind = "tool"
		appendApprovedStep(meta.Inputs, "_approvedDangerousToolSteps", step.StepID)
	} else {
		appendApprovedStep(meta.Inputs, "_approvedHumanSteps", step.StepID)
	}
	if err := saveRunMeta(s.db.RunDir(runID), *meta); err != nil {
		return nil, err
	}

	_, _ = s.events.Append(runID, rec.TraceID, "gate.approved", "info", map[string]any{
		"actorId": req.ActorID,
		"reason":  req.Reason,
		"stepId":  step.StepID,
		"kind":    approvalKind,
	})
	_ = s.writeAudit(runID, rec.TraceID, "gate.approved", map[string]any{
		"actorId": req.ActorID,
		"reason":  req.Reason,
		"stepId":  step.StepID,
		"kind":    approvalKind,
	})
	s.decidePendingApproval(runID, step.StepID, "approved", req.ActorID, req.Reason)

	rec.Status = "running"
	rec.FinishedAt = nil
	rec.ErrorCode = ""
	rec.ErrorMessage = ""
	rec.UpdatedAt = time.Now().UTC()
	if err := s.db.Save(&rec).Error; err != nil {
		return nil, err
	}
	_, _ = s.events.Append(runID, rec.TraceID, "run.approval_resumed", "info", map[string]any{
		"fromStatus": "waiting_approval",
		"stepId":     step.StepID,
		"kind":       approvalKind,
	})
	_ = s.writeAudit(runID, rec.TraceID, "run.approval_resumed", map[string]any{
		"fromStatus": "waiting_approval",
		"stepId":     step.StepID,
		"kind":       approvalKind,
	})

	doc, err := s.scenarios.Get(rec.ScenarioName, rec.ScenarioVersion)
	if err != nil {
		return nil, err
	}
	eng := rules.NewEngine(doc)
	createReq := CreateRequest{
		Scenario:      meta.Scenario,
		Inputs:        meta.Inputs,
		PolicyProfile: meta.PolicyProfile,
		Repo:          meta.Repo,
		SpaceID:       rec.SpaceID,
	}
	if err := s.executeSteps(&rec, createReq, doc, eng, rec.StartedAt); err != nil {
		if errors.Is(err, ErrWaitingApproval) {
			return &ApproveResponse{RunID: runID, OK: true}, nil
		}
		return nil, err
	}
	return &ApproveResponse{RunID: runID, OK: true}, nil
}

func (s *Service) RAG() *rag.Service       { return s.rag }
func (s *Service) DB() *store.DB           { return s.db }
func (s *Service) Events() *events.Service { return s.events }

func recordToSummary(rec store.RunRecord) *Summary {
	sum := &Summary{
		RunID: rec.ID, TraceID: rec.TraceID,
		Scenario:      ScenarioRef{Name: rec.ScenarioName, ScenarioVersion: rec.ScenarioVersion},
		PolicyProfile: rec.PolicyProfile, Status: rec.Status,
		StartedAt: rec.StartedAt.UnixMilli(), Recovered: rec.Recovered, InputsDigest: rec.InputsDigest,
		SpaceID: rec.SpaceID, ActorRole: rec.ActorRole,
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

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
