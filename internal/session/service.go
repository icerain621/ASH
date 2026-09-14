package session

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
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
	ID               string         `json:"id"`
	SpaceID          string         `json:"spaceId"`
	Status           string         `json:"status"`
	Goal             string         `json:"goal,omitempty"`
	PlanID           string         `json:"planId,omitempty"`
	RunID            string         `json:"runId,omitempty"`
	TraceID          string         `json:"traceId,omitempty"`
	RepoRoot         string         `json:"repoRoot,omitempty"`
	StreamURL        string         `json:"streamUrl,omitempty"`
	ProviderKind     string         `json:"providerKind,omitempty"`
	ProviderAdapter  string         `json:"providerAdapter,omitempty"`
	ProviderFallback bool           `json:"providerFallback,omitempty"`
	ProviderReason   string         `json:"providerReason,omitempty"`
	Turns            []Turn         `json:"turns"`
	CreatedBy        string         `json:"createdBy,omitempty"`
	CreatedAt        int64          `json:"createdAt"`
	UpdatedAt        int64          `json:"updatedAt"`
	Meta             map[string]any `json:"meta,omitempty"`
}

type Turn struct {
	ID        string `json:"id"`
	Prompt    string `json:"prompt"`
	CreatedAt int64  `json:"createdAt"`
}

type CreateRequest struct {
	Goal         string `json:"goal"`
	RunID        string `json:"runId"`
	RepoRoot     string `json:"repoRoot"`
	SpaceID      string `json:"spaceId"`
	ActorRole    string `json:"actorRole"`
	CreatedBy    string `json:"createdBy"`
	AutoApprove  bool   `json:"autoApprove"`
	ProviderKind string `json:"providerKind"`
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

// ListResponse is the session list envelope for API/swagger.
type ListResponse struct {
	Items []View `json:"items"`
}

type Service struct {
	db     *store.DB
	goal   *goal.Service
	events *events.Service
	runs   RunControl
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
	// Do not call goal.WithContext here: goal → runs → session forms a cycle.
	if s.events != nil {
		out.events = s.events.WithContext(ctx)
	}
	return &out
}

func (s *Service) goalFor() *goal.Service {
	if s == nil || s.goal == nil {
		return nil
	}
	if s.ctx != nil {
		return s.goal.WithContext(s.ctx)
	}
	return s.goal
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
		goalSvc := s.goalFor()
		if goalSvc == nil {
			return nil, fmt.Errorf("goal service is not configured")
		}
		view.Goal = goalText
		plan, err := goalSvc.FromGoal(goal.FromGoalRequest{
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
	s.applyProviderKind(view, req.ProviderKind)
	s.ensureMainThread(view)
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

// List returns agent sessions for a space, newest UpdatedAt/CreatedAt first.
func (s *Service) List(spaceID string, limit int) ([]View, error) {
	spaceID = firstNonEmpty(strings.TrimSpace(spaceID), "local")
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	var rows []store.AuditLog
	if err := s.q().Where("event_type = ? AND space_id = ?", auditEventType, spaceID).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]View, 0, len(rows))
	for _, row := range rows {
		view, err := decodeView(row)
		if err != nil {
			continue
		}
		view.StreamURL = streamURL(view.RunID)
		if view.CreatedAt == 0 && !row.CreatedAt.IsZero() {
			view.CreatedAt = row.CreatedAt.Unix()
		}
		if view.UpdatedAt == 0 {
			view.UpdatedAt = view.CreatedAt
		}
		out = append(out, *view)
	}
	sort.Slice(out, func(i, j int) bool {
		ai := out[i].UpdatedAt
		if ai == 0 {
			ai = out[i].CreatedAt
		}
		aj := out[j].UpdatedAt
		if aj == 0 {
			aj = out[j].CreatedAt
		}
		if ai != aj {
			return ai > aj
		}
		return out[i].ID > out[j].ID
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// PromptTurn records a turn and emits session.turn on the bound run when present.
// When providerKind=acp_sdk and ACP is healthy, best-effort forwards the prompt to ACP (DX4).
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

	acpPayload := s.forwardTurnACP(view, turn)

	if view.RunID != "" && s.events != nil {
		trace := firstNonEmpty(view.TraceID, view.RunID)
		payload := map[string]any{
			"sessionId": view.ID, "turnId": turn.ID, "prompt": prompt,
		}
		for k, v := range acpPayload {
			payload[k] = v
		}
		_, _ = s.events.Append(view.RunID, trace, "session.turn", "info", payload)
	}
	if err := s.save(view); err != nil {
		return nil, nil, err
	}
	view.StreamURL = streamURL(view.RunID)
	return view, &turn, nil
}

// ListEvents returns recent run events for the session's bound run.
// When the session has no runId, turns are projected as session.turn envelopes.
func (s *Service) ListEvents(sessionID string, afterSeq int64, limit int) (EventsResponse, error) {
	view, err := s.Get(sessionID)
	if err != nil {
		return EventsResponse{}, err
	}
	out := EventsResponse{
		SessionID: view.ID, RunID: view.RunID, StreamURL: streamURL(view.RunID), Items: []events.Envelope{},
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if view.RunID == "" {
		out.Items = synthesizeTurnEvents(view, afterSeq, limit)
		return out, nil
	}
	if s.events == nil {
		return out, nil
	}
	items, err := s.events.ListAfter(view.RunID, afterSeq, limit)
	if err != nil {
		return EventsResponse{}, err
	}
	out.Items = items
	return out, nil
}

func synthesizeTurnEvents(view *View, afterSeq int64, limit int) []events.Envelope {
	if view == nil || len(view.Turns) == 0 {
		return []events.Envelope{}
	}
	out := make([]events.Envelope, 0, len(view.Turns))
	for i, turn := range view.Turns {
		seq := int64(i + 1)
		if seq <= afterSeq {
			continue
		}
		payload, _ := json.Marshal(map[string]any{"prompt": turn.Prompt})
		ts := turn.CreatedAt
		if ts > 0 && ts < 1_000_000_000_000 {
			ts = ts * 1000
		}
		id := turn.ID
		if id == "" {
			id = fmt.Sprintf("turn_seq_%d", seq)
		}
		out = append(out, events.Envelope{
			ID:         id,
			TraceID:    view.TraceID,
			RunID:      view.RunID,
			Seq:        seq,
			TS:         ts,
			Type:       "session.turn",
			Severity:   "info",
			Visibility: events.VisibilityModelVisible,
			Payload:    payload,
		})
		if len(out) >= limit {
			break
		}
	}
	return out
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
