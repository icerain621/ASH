package session

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ash-repwiki/ash/internal/events"
	"github.com/ash-repwiki/ash/internal/goal"
	"github.com/ash-repwiki/ash/internal/store"
)

const (
	StatusActive = "active"
	StatusClosed = "closed"

	auditEventType = "agent.session"
)

// View is the public session document.
type View struct {
	ID        string         `json:"id"`
	SpaceID   string         `json:"spaceId"`
	Status    string         `json:"status"`
	Goal      string         `json:"goal,omitempty"`
	PlanID    string         `json:"planId,omitempty"`
	RunID     string         `json:"runId,omitempty"`
	TraceID   string         `json:"traceId,omitempty"`
	RepoRoot  string         `json:"repoRoot,omitempty"`
	StreamURL string         `json:"streamUrl,omitempty"`
	Turns     []Turn         `json:"turns"`
	CreatedBy string         `json:"createdBy,omitempty"`
	CreatedAt int64          `json:"createdAt"`
	UpdatedAt int64          `json:"updatedAt"`
	Meta      map[string]any `json:"meta,omitempty"`
}

type Turn struct {
	ID        string `json:"id"`
	Prompt    string `json:"prompt"`
	CreatedAt int64  `json:"createdAt"`
}

type CreateRequest struct {
	Goal        string `json:"goal"`
	RunID       string `json:"runId"`
	RepoRoot    string `json:"repoRoot"`
	SpaceID     string `json:"spaceId"`
	ActorRole   string `json:"actorRole"`
	CreatedBy   string `json:"createdBy"`
	AutoApprove bool   `json:"autoApprove"`
}

type TurnRequest struct {
	Prompt string `json:"prompt" binding:"required"`
}

type EventsResponse struct {
	SessionID string            `json:"sessionId"`
	RunID     string            `json:"runId,omitempty"`
	StreamURL string            `json:"streamUrl,omitempty"`
	Items     []events.Envelope `json:"items"`
}

type Service struct {
	db     *store.DB
	goal   *goal.Service
	events *events.Service
	ctx    context.Context
}

func NewService(db *store.DB, goalSvc *goal.Service, ev *events.Service) *Service {
	return &Service{db: db, goal: goalSvc, events: ev}
}

func (s *Service) WithContext(ctx context.Context) *Service {
	if s == nil || ctx == nil {
		return s
	}
	out := *s
	out.ctx = ctx
	if s.db != nil {
		out.db = s.db.BindContext(ctx)
	}
	if s.goal != nil {
		out.goal = s.goal.WithContext(ctx)
	}
	if s.events != nil {
		out.events = s.events.WithContext(ctx)
	}
	return &out
}

func (s *Service) q() *gorm.DB {
	if s.ctx != nil && s.db != nil {
		return s.db.WithContext(s.ctx)
	}
	return s.db.DB
}

// Create starts a session, optionally binding a run or routing a goal.
func (s *Service) Create(req CreateRequest) (*View, error) {
	space := firstNonEmpty(strings.TrimSpace(req.SpaceID), "local")
	now := time.Now().UTC()
	id := "sess_" + uuid.NewString()
	view := &View{
		ID: id, SpaceID: space, Status: StatusActive,
		RepoRoot:  strings.TrimSpace(req.RepoRoot),
		CreatedBy: strings.TrimSpace(req.CreatedBy),
		Turns:     []Turn{}, CreatedAt: now.Unix(), UpdatedAt: now.Unix(),
	}

	runID := strings.TrimSpace(req.RunID)
	goalText := strings.TrimSpace(req.Goal)
	if runID != "" && goalText != "" {
		return nil, fmt.Errorf("provide either runId or goal, not both")
	}
	if runID != "" {
		var rec store.RunRecord
		if err := s.q().First(&rec, "id = ? AND space_id = ?", runID, space).Error; err != nil {
			return nil, fmt.Errorf("run not found: %w", err)
		}
		view.RunID = rec.ID
		view.TraceID = rec.TraceID
		if view.RepoRoot == "" {
			view.RepoRoot = rec.RepoRoot
		}
	} else if goalText != "" {
		if s.goal == nil {
			return nil, fmt.Errorf("goal service is not configured")
		}
		view.Goal = goalText
		plan, err := s.goal.FromGoal(goal.FromGoalRequest{
			Goal: goalText, RepoRoot: firstNonEmpty(view.RepoRoot, "."),
			SpaceID: space, ActorRole: firstNonEmpty(req.ActorRole, "maintainer"),
			CreatedBy: firstNonEmpty(req.CreatedBy, "session"), AutoApprove: req.AutoApprove,
		})
		if err != nil && plan == nil {
			return nil, err
		}
		if plan != nil {
			view.PlanID = plan.ID
			view.RunID = plan.RunID
			view.TraceID = plan.TraceID
			view.Meta = map[string]any{
				"scenarioName": plan.ScenarioName, "scenarioVersion": plan.ScenarioVersion,
				"planStatus": plan.Status, "routeReason": plan.RouteReason,
			}
			if err != nil {
				view.Meta["executionError"] = err.Error()
			}
		}
	}
	view.StreamURL = streamURL(view.RunID)
	if err := s.save(view); err != nil {
		return nil, err
	}
	return view, nil
}

func (s *Service) Get(sessionID string) (*View, error) {
	row, err := s.loadRow(sessionID)
	if err != nil {
		return nil, err
	}
	view, err := decodeView(row)
	if err != nil {
		return nil, err
	}
	view.StreamURL = streamURL(view.RunID)
	return view, nil
}

// PromptTurn records a turn and emits session.turn on the bound run when present.
func (s *Service) PromptTurn(sessionID string, req TurnRequest) (*View, *Turn, error) {
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		return nil, nil, fmt.Errorf("prompt is required")
	}
	view, err := s.Get(sessionID)
	if err != nil {
		return nil, nil, err
	}
	if view.Status != StatusActive {
		return nil, nil, fmt.Errorf("session status %q cannot accept turns", view.Status)
	}
	turn := Turn{
		ID: "turn_" + uuid.NewString(), Prompt: prompt, CreatedAt: time.Now().UTC().Unix(),
	}
	view.Turns = append(view.Turns, turn)
	view.UpdatedAt = turn.CreatedAt
	if view.RunID != "" && s.events != nil {
		trace := firstNonEmpty(view.TraceID, view.RunID)
		_, _ = s.events.Append(view.RunID, trace, "session.turn", "info", map[string]any{
			"sessionId": view.ID, "turnId": turn.ID, "prompt": prompt,
		})
	}
	if err := s.save(view); err != nil {
		return nil, nil, err
	}
	view.StreamURL = streamURL(view.RunID)
	return view, &turn, nil
}

// ListEvents returns recent run events for the session's bound run.
func (s *Service) ListEvents(sessionID string, afterSeq int64, limit int) (EventsResponse, error) {
	view, err := s.Get(sessionID)
	if err != nil {
		return EventsResponse{}, err
	}
	out := EventsResponse{
		SessionID: view.ID, RunID: view.RunID, StreamURL: streamURL(view.RunID), Items: []events.Envelope{},
	}
	if view.RunID == "" || s.events == nil {
		return out, nil
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	items, err := s.events.ListAfter(view.RunID, afterSeq, limit)
	if err != nil {
		return EventsResponse{}, err
	}
	out.Items = items
	return out, nil
}

func (s *Service) loadRow(sessionID string) (store.AuditLog, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return store.AuditLog{}, fmt.Errorf("sessionId is required")
	}
	var row store.AuditLog
	if err := s.q().First(&row, "id = ? AND event_type = ?", sessionID, auditEventType).Error; err != nil {
		return store.AuditLog{}, fmt.Errorf("session not found: %w", err)
	}
	return row, nil
}

func (s *Service) save(view *View) error {
	if view == nil {
		return fmt.Errorf("session view is nil")
	}
	payload, err := json.Marshal(view)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	row := store.AuditLog{
		ID: view.ID, SpaceID: view.SpaceID, TraceID: view.TraceID, RunID: view.RunID,
		ActorID: firstNonEmpty(view.CreatedBy, "session"), EventType: auditEventType,
		PayloadJSON: string(payload), CreatedAt: time.Unix(view.CreatedAt, 0).UTC(),
	}
	if row.CreatedAt.IsZero() {
		row.CreatedAt = now
	}
	var existing store.AuditLog
	err = s.q().First(&existing, "id = ?", view.ID).Error
	if err == gorm.ErrRecordNotFound {
		return s.q().Create(&row).Error
	}
	if err != nil {
		return err
	}
	return s.q().Model(&store.AuditLog{}).Where("id = ?", view.ID).Updates(map[string]any{
		"space_id": view.SpaceID, "trace_id": view.TraceID, "run_id": view.RunID,
		"actor_id": row.ActorID, "payload_json": string(payload),
	}).Error
}

func decodeView(row store.AuditLog) (*View, error) {
	var view View
	if err := json.Unmarshal([]byte(row.PayloadJSON), &view); err != nil {
		return nil, fmt.Errorf("decode session: %w", err)
	}
	if view.ID == "" {
		view.ID = row.ID
	}
	if view.Turns == nil {
		view.Turns = []Turn{}
	}
	return &view, nil
}

func streamURL(runID string) string {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return ""
	}
	return "/api/v1/runs/" + runID + "/stream"
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
