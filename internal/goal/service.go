package goal

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/ash-repwiki/ash/internal/events"
	"github.com/ash-repwiki/ash/internal/rules"
	"github.com/ash-repwiki/ash/internal/runs"
	"github.com/ash-repwiki/ash/internal/spacerules"
	"github.com/ash-repwiki/ash/internal/store"
)

const (
	StatusDraft    = "draft"
	StatusApproved = "approved"
	StatusRejected = "rejected"
	StatusStarted  = "started"
)

type PlanView struct {
	ID              string         `json:"id"`
	SpaceID         string         `json:"spaceId"`
	Goal            string         `json:"goal"`
	ScenarioName    string         `json:"scenarioName"`
	ScenarioVersion string         `json:"scenarioVersion"`
	PolicyProfile   string         `json:"policyProfile"`
	RouteReason     string         `json:"routeReason"`
	Inputs          map[string]any `json:"inputs"`
	Steps           []StepPreview  `json:"steps"`
	Status          string         `json:"status"`
	RunID           string         `json:"runId,omitempty"`
	TraceID         string         `json:"traceId,omitempty"`
	CreatedBy       string         `json:"createdBy,omitempty"`
	CreatedAt       int64          `json:"createdAt"`
	UpdatedAt       int64          `json:"updatedAt"`
}

type FromGoalRequest struct {
	Goal         string `json:"goal"`
	RepoRoot     string `json:"repoRoot"`
	SpaceID      string `json:"spaceId"`
	ActorRole    string `json:"actorRole"`
	CreatedBy    string `json:"createdBy"`
	AutoApprove  bool   `json:"autoApprove"`
	PolicyProfile string `json:"policyProfile"`
}

type Service struct {
	db        *store.DB
	scenarios *rules.Loader
	runs      *runs.Service
	events    *events.Service
	rules     *spacerules.Service
}

func NewService(db *store.DB, scenarios *rules.Loader, runsSvc *runs.Service, ev *events.Service) *Service {
	return &Service{
		db: db, scenarios: scenarios, runs: runsSvc, events: ev,
		rules: spacerules.NewService(db),
	}
}

func (s *Service) WithContext(ctx context.Context) *Service {
	if s == nil || ctx == nil {
		return s
	}
	out := *s
	out.db = s.db.BindContext(ctx)
	if s.runs != nil {
		out.runs = s.runs.WithContext(ctx)
	}
	if s.events != nil {
		out.events = s.events.WithContext(ctx)
	}
	return &out
}

func (s *Service) FromGoal(req FromGoalRequest) (*PlanView, error) {
	space := firstNonEmpty(strings.TrimSpace(req.SpaceID), "local")
	rulesDoc := spacerules.BuiltinDocument()
	if s.rules != nil {
		if view, err := s.rules.Get(space); err == nil && view != nil {
			rulesDoc = view.Document
		}
	}
	routed, err := RouteWithDoc(req.Goal, s.scenarios, req.RepoRoot, rulesDoc)
	if err != nil {
		return nil, err
	}
	if req.PolicyProfile != "" {
		routed.PolicyProfile = req.PolicyProfile
	}
	inputsJSON, err := json.Marshal(routed.Inputs)
	if err != nil {
		return nil, err
	}
	stepsJSON, err := json.Marshal(routed.Steps)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	row := store.GoalPlan{
		ID: "gplan_" + uuid.NewString(), SpaceID: space, Goal: strings.TrimSpace(req.Goal),
		ScenarioName: routed.ScenarioName, ScenarioVersion: routed.ScenarioVersion,
		PolicyProfile: routed.PolicyProfile, RouteReason: routed.Reason,
		InputsJSON: string(inputsJSON), StepsJSON: string(stepsJSON),
		Status: StatusDraft, CreatedBy: strings.TrimSpace(req.CreatedBy),
		CreatedAt: now, UpdatedAt: now,
	}
	if err := s.db.Create(&row).Error; err != nil {
		return nil, err
	}
	view := toView(row)
	_ = s.emitPlan(row.ID, row.ID, "plan.created", map[string]any{
		"planId": row.ID, "scenario": row.ScenarioName, "version": row.ScenarioVersion, "reason": row.RouteReason,
	})
	if req.AutoApprove {
		return s.Approve(row.ID, firstNonEmpty(req.CreatedBy, "system"), "autoApprove", firstNonEmpty(req.ActorRole, "maintainer"))
	}
	return view, nil
}

func (s *Service) Get(planID string) (*PlanView, error) {
	var row store.GoalPlan
	if err := s.db.First(&row, "id = ?", planID).Error; err != nil {
		return nil, err
	}
	return toView(row), nil
}

func (s *Service) Approve(planID, actorID, reason, actorRole string) (*PlanView, error) {
	var row store.GoalPlan
	if err := s.db.First(&row, "id = ?", planID).Error; err != nil {
		return nil, err
	}
	if row.Status != StatusDraft {
		return nil, fmt.Errorf("plan status %q cannot be approved", row.Status)
	}
	var inputs map[string]any
	if err := json.Unmarshal([]byte(row.InputsJSON), &inputs); err != nil {
		return nil, err
	}
	_ = s.emitPlan(row.ID, row.ID, "plan.approved", map[string]any{
		"planId": row.ID, "actorId": actorID, "reason": reason,
	})
	resp, err := s.runs.Create(runs.CreateRequest{
		Scenario:      runs.ScenarioRef{Name: row.ScenarioName, ScenarioVersion: row.ScenarioVersion},
		Inputs:        inputs,
		PolicyProfile: row.PolicyProfile,
		SpaceID:       row.SpaceID,
		ActorRole:     firstNonEmpty(actorRole, "maintainer"),
		Repo:          &runs.RepoRef{Root: inputString(inputs, "repoRoot")},
	})
	if err != nil && resp == nil {
		return nil, err
	}
	now := time.Now().UTC()
	row.Status = StatusStarted
	row.RunID = resp.RunID
	row.UpdatedAt = now
	if err := s.db.Save(&row).Error; err != nil {
		return nil, err
	}
	_ = s.emitPlan(resp.RunID, resp.TraceID, "plan.started", map[string]any{
		"planId": row.ID, "runId": resp.RunID,
	})
	view := toView(row)
	view.TraceID = resp.TraceID
	if err != nil {
		// Run created but execution reported error — still return plan with runId.
		return view, err
	}
	return view, nil
}

func (s *Service) Reject(planID, actorID, reason string) (*PlanView, error) {
	var row store.GoalPlan
	if err := s.db.First(&row, "id = ?", planID).Error; err != nil {
		return nil, err
	}
	if row.Status != StatusDraft {
		return nil, fmt.Errorf("plan status %q cannot be rejected", row.Status)
	}
	row.Status = StatusRejected
	row.UpdatedAt = time.Now().UTC()
	if err := s.db.Save(&row).Error; err != nil {
		return nil, err
	}
	_ = s.emitPlan(row.ID, row.ID, "plan.rejected", map[string]any{
		"planId": row.ID, "actorId": actorID, "reason": reason,
	})
	return toView(row), nil
}

func (s *Service) emitPlan(runID, traceID, typ string, payload map[string]any) error {
	if s == nil || s.events == nil {
		return nil
	}
	_, err := s.events.Append(runID, traceID, typ, "info", payload)
	return err
}

func toView(row store.GoalPlan) *PlanView {
	var inputs map[string]any
	_ = json.Unmarshal([]byte(row.InputsJSON), &inputs)
	var steps []StepPreview
	_ = json.Unmarshal([]byte(row.StepsJSON), &steps)
	return &PlanView{
		ID: row.ID, SpaceID: row.SpaceID, Goal: row.Goal,
		ScenarioName: row.ScenarioName, ScenarioVersion: row.ScenarioVersion,
		PolicyProfile: row.PolicyProfile, RouteReason: row.RouteReason,
		Inputs: inputs, Steps: steps, Status: row.Status, RunID: row.RunID,
		CreatedBy: row.CreatedBy,
		CreatedAt: row.CreatedAt.UnixMilli(), UpdatedAt: row.UpdatedAt.UnixMilli(),
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func inputString(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}
