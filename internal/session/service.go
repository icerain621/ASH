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
	"github.com/ash-repwiki/ash/internal/llmchat"
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
	Title            string         `json:"title,omitempty"`
	Goal             string         `json:"goal,omitempty"`
	PlanID           string         `json:"planId,omitempty"`
	RunID            string         `json:"runId,omitempty"`
	TraceID          string         `json:"traceId,omitempty"`
	RepoRoot         string         `json:"repoRoot,omitempty"`
	WorkspaceID      string         `json:"workspaceId,omitempty"`
	StreamURL        string         `json:"streamUrl,omitempty"`
	ProviderKind     string         `json:"providerKind,omitempty"`
	ProviderAdapter  string         `json:"providerAdapter,omitempty"`
	ProviderFallback bool           `json:"providerFallback,omitempty"`
	ProviderReason   string         `json:"providerReason,omitempty"`
	PermissionMode   string         `json:"permissionMode,omitempty"` // read-only | workspace-write | full
	AgentMode        string         `json:"agentMode,omitempty"`      // coding | general
	DisabledTools    []string       `json:"disabledTools,omitempty"`
	Turns            []Turn           `json:"turns"`
	Replies          []AssistantReply `json:"replies,omitempty"` // blank-session assistant prose (no runId)
	CreatedBy        string           `json:"createdBy,omitempty"`
	CreatedAt        int64            `json:"createdAt"`
	UpdatedAt        int64            `json:"updatedAt"`
	Meta             map[string]any   `json:"meta,omitempty"`
}

// PatchRequest updates mutable session seat fields.
type PatchRequest struct {
	Title          *string   `json:"title"`
	ProviderKind   *string   `json:"providerKind"`
	PlanID         *string   `json:"planId"`
	PermissionMode *string   `json:"permissionMode"`
	AgentMode      *string   `json:"agentMode"`
	DisabledTools  *[]string `json:"disabledTools"`
}

const maxTitleRunes = 48

type Turn struct {
	ID        string `json:"id"`
	Prompt    string `json:"prompt"`
	CreatedAt int64  `json:"createdAt"`
}

type CreateRequest struct {
	Goal         string `json:"goal"`
	RunID        string `json:"runId"`
	RepoRoot     string `json:"repoRoot"`
	WorkspaceID  string `json:"workspaceId"`
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
	db      *store.DB
	goal    *goal.Service
	events  *events.Service
	runs    RunControl
	ctx     context.Context
	flights *flightRegistry
}

func NewService(db *store.DB, goalSvc *goal.Service, ev *events.Service) *Service {
	return &Service{db: db, goal: goalSvc, events: ev, flights: newFlightRegistry()}
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
		RepoRoot:    strings.TrimSpace(req.RepoRoot),
		WorkspaceID: strings.TrimSpace(req.WorkspaceID),
		CreatedBy:   strings.TrimSpace(req.CreatedBy),
		Turns:       []Turn{}, Replies: []AssistantReply{}, CreatedAt: now.Unix(), UpdatedAt: now.Unix(),
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
		if view.Title == "" {
			view.Title = truncateTitle(goalText, maxTitleRunes)
		}
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
	view.StreamURL = sessionStreamURL(view.ID)
	s.applyProviderKind(view, req.ProviderKind)
	view.AgentMode = AgentModeCoding
	if view.Meta == nil {
		view.Meta = map[string]any{}
	}
	view.Meta["agentMode"] = AgentModeCoding
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
	view.StreamURL = sessionStreamURL(view.ID)
	return view, nil
}

// List returns agent sessions for a space, newest UpdatedAt/CreatedAt first.
// By default closed sessions are excluded; pass includeClosed=true to include them.
func (s *Service) List(spaceID string, limit int, includeClosed bool) ([]View, error) {
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
		if !includeClosed && view.Status == StatusClosed {
			continue
		}
		view.StreamURL = sessionStreamURL(view.ID)
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
// Reply priority: OpenAI-compatible LLM (ASH_LLM_BASE_URL) → provider executor → echo stub.
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
	if view.Title == "" {
		view.Title = truncateTitle(prompt, maxTitleRunes)
	}

	flightCtx, flight := s.flights.begin(sessionID)
	defer s.flights.end(sessionID, flight)
	flight.turnID = turn.ID

	usedLLM := false
	var providerPayload map[string]any

	if llmchat.Configured() {
		usedLLM = s.replyViaLLM(flightCtx, flight, view, turn, prompt, func(extra map[string]any) {
			s.emitSessionTurn(view, turn, prompt, extra)
		})
	}

	if !usedLLM {
		if err := flightCtx.Err(); err != nil {
			s.emitAssistantReplyChunks(view, turn, "（已停止）", "echo", []string{"（已停止）"}, true)
		} else {
			providerPayload = s.forwardTurnProvider(view, turn)
			s.emitSessionTurn(view, turn, prompt, providerPayload)
			replyText, replySource := resolveAssistantText(prompt, providerPayload)
			if err := flightCtx.Err(); err != nil {
				s.emitAssistantReplyChunks(view, turn, replyText, replySource, splitReplyChunks(replyText), true)
			} else {
				s.emitAssistantReply(view, turn, replyText, replySource)
			}
		}
	}

	if err := s.save(view); err != nil {
		return nil, nil, err
	}
	view.StreamURL = sessionStreamURL(view.ID)
	return view, &turn, nil
}

func (s *Service) emitSessionTurn(view *View, turn Turn, prompt string, extra map[string]any) {
	if view == nil || strings.TrimSpace(view.RunID) == "" || s.events == nil {
		return
	}
	trace := firstNonEmpty(view.TraceID, view.RunID)
	payload := map[string]any{
		"sessionId": view.ID, "turnId": turn.ID, "prompt": prompt,
	}
	for k, v := range extra {
		payload[k] = v
	}
	_, _ = s.events.Append(view.RunID, trace, "session.turn", "info", payload)
}

// Update applies a partial patch (title / providerKind / planId / permissionMode / agentMode).
func (s *Service) Update(sessionID string, req PatchRequest) (*View, error) {
	view, err := s.Get(sessionID)
	if err != nil {
		return nil, err
	}
	if req.Title != nil {
		view.Title = truncateTitle(strings.TrimSpace(*req.Title), maxTitleRunes)
	}
	if req.ProviderKind != nil {
		s.applyProviderKind(view, *req.ProviderKind)
	}
	if req.PlanID != nil {
		view.PlanID = strings.TrimSpace(*req.PlanID)
	}
	if req.PermissionMode != nil {
		mode, err := normalizePermissionMode(*req.PermissionMode)
		if err != nil {
			return nil, err
		}
		view.PermissionMode = mode
		if view.Meta == nil {
			view.Meta = map[string]any{}
		}
		if mode != "" {
			view.Meta["permissionMode"] = mode
		} else {
			delete(view.Meta, "permissionMode")
		}
	}
	if req.AgentMode != nil {
		mode, err := normalizeAgentMode(*req.AgentMode)
		if err != nil {
			return nil, err
		}
		view.AgentMode = mode
		if view.Meta == nil {
			view.Meta = map[string]any{}
		}
		view.Meta["agentMode"] = mode
	}
	if req.DisabledTools != nil {
		cleaned := normalizeDisabledTools(*req.DisabledTools)
		view.DisabledTools = cleaned
		if view.Meta == nil {
			view.Meta = map[string]any{}
		}
		if len(cleaned) == 0 {
			delete(view.Meta, "disabledTools")
		} else {
			view.Meta["disabledTools"] = cleaned
		}
	}
	view.UpdatedAt = time.Now().UTC().Unix()
	if err := s.save(view); err != nil {
		return nil, err
	}
	view.StreamURL = sessionStreamURL(view.ID)
	return view, nil
}

// SetWorkspaceID records the workspace membership on the session document.
func (s *Service) SetWorkspaceID(sessionID, workspaceID string) (*View, error) {
	view, err := s.Get(sessionID)
	if err != nil {
		return nil, err
	}
	view.WorkspaceID = strings.TrimSpace(workspaceID)
	view.UpdatedAt = time.Now().UTC().Unix()
	if err := s.save(view); err != nil {
		return nil, err
	}
	view.StreamURL = sessionStreamURL(view.ID)
	return view, nil
}

// Close soft-closes a session (status=closed). Turns are rejected afterwards.
func (s *Service) Close(sessionID string) (*View, error) {
	view, err := s.Get(sessionID)
	if err != nil {
		return nil, err
	}
	if view.Status == StatusClosed {
		view.StreamURL = sessionStreamURL(view.ID)
		return view, nil
	}
	view.Status = StatusClosed
	view.UpdatedAt = time.Now().UTC().Unix()
	if err := s.save(view); err != nil {
		return nil, err
	}
	view.StreamURL = sessionStreamURL(view.ID)
	return view, nil
}

// PurgeResult is returned after hard-deleting a session audit row.
type PurgeResult struct {
	ID     string `json:"id"`
	Purged bool   `json:"purged"`
}

// Purge permanently deletes the agent.session audit_log row (hard delete).
// Bound run ledger events are left intact; Get/List will no longer find the session.
func (s *Service) Purge(sessionID string) (*PurgeResult, error) {
	row, err := s.loadRow(sessionID)
	if err != nil {
		return nil, err
	}
	if err := s.q().Delete(&store.AuditLog{}, "id = ? AND event_type = ?", row.ID, auditEventType).Error; err != nil {
		return nil, fmt.Errorf("purge session: %w", err)
	}
	return &PurgeResult{ID: row.ID, Purged: true}, nil
}

// ListEvents returns recent run events for the session's bound run.
// When the session has no runId, turns are projected as session.turn envelopes.
func (s *Service) ListEvents(sessionID string, afterSeq int64, limit int) (EventsResponse, error) {
	view, err := s.Get(sessionID)
	if err != nil {
		return EventsResponse{}, err
	}
	out := EventsResponse{
		SessionID: view.ID, RunID: view.RunID, StreamURL: sessionStreamURL(view.ID), Items: []events.Envelope{},
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if view.RunID == "" {
		out.Items = synthesizeTurnEvents(view, afterSeq, limit)
		if s.flights != nil {
			if turnID, text, ok := s.flights.partial(sessionID); ok {
				out.Items = appendLivePartial(out.Items, view, turnID, text, afterSeq)
			}
		}
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
	repliesByTurn := make(map[string]AssistantReply, len(view.Replies))
	for _, r := range view.Replies {
		repliesByTurn[r.TurnID] = r
	}
	out := make([]events.Envelope, 0, len(view.Turns)*4)
	var seq int64
	appendEv := func(id, typ string, ts int64, payload map[string]any) bool {
		seq++
		if seq <= afterSeq {
			return false
		}
		if len(out) >= limit {
			return true
		}
		raw, _ := json.Marshal(payload)
		if ts > 0 && ts < 1_000_000_000_000 {
			ts = ts * 1000
		}
		if id == "" {
			id = fmt.Sprintf("%s_seq_%d", typ, seq)
		}
		out = append(out, events.Envelope{
			ID:         id,
			TraceID:    view.TraceID,
			RunID:      view.RunID,
			Seq:        seq,
			TS:         ts,
			Type:       typ,
			Severity:   "info",
			Visibility: events.VisibilityModelVisible,
			Payload:    raw,
		})
		return len(out) >= limit
	}
	for _, turn := range view.Turns {
		ts := turn.CreatedAt
		id := turn.ID
		if appendEv(id, "session.turn", ts, map[string]any{"prompt": turn.Prompt, "turnId": turn.ID}) {
			break
		}
		reply, ok := repliesByTurn[turn.ID]
		if !ok {
			continue
		}
		chunks := reply.Chunks
		if len(chunks) == 0 {
			chunks = splitReplyChunks(reply.Text)
		}
		full := false
		for i, chunk := range chunks {
			deltaID := fmt.Sprintf("%s_delta_%d", turn.ID, i)
			if appendEv(deltaID, "assistant.delta", ts, map[string]any{
				"turnId": turn.ID, "text": chunk, "index": i,
			}) {
				full = true
				break
			}
		}
		if full {
			break
		}
		msgID := turn.ID + "_assistant"
		if appendEv(msgID, "assistant.message", ts, map[string]any{
			"turnId": turn.ID, "text": reply.Text, "stopped": reply.Stopped, "source": reply.Source,
		}) {
			break
		}
	}
	return out
}

// appendLivePartial overlays an in-flight assistant.delta for blank-session SSE polling.
func appendLivePartial(items []events.Envelope, view *View, turnID, text string, afterSeq int64) []events.Envelope {
	if view == nil || strings.TrimSpace(text) == "" {
		return items
	}
	var maxSeq int64
	for _, ev := range items {
		if ev.Seq > maxSeq {
			maxSeq = ev.Seq
		}
	}
	seq := maxSeq + 1
	if seq <= afterSeq {
		return items
	}
	raw, _ := json.Marshal(map[string]any{"turnId": turnID, "text": text, "index": 0, "streaming": true})
	ts := time.Now().UTC().UnixMilli()
	items = append(items, events.Envelope{
		ID: turnID + "_live", RunID: "", Seq: seq, TS: ts,
		Type: "assistant.delta", Severity: "info", Visibility: events.VisibilityModelVisible,
		Payload: raw,
	})
	return items
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
	if view.Replies == nil {
		view.Replies = []AssistantReply{}
	}
	if len(view.DisabledTools) == 0 && view.Meta != nil {
		if raw, ok := view.Meta["disabledTools"]; ok {
			view.DisabledTools = coerceStringSlice(raw)
		}
	}
	hydrateAgentMode(&view)
	return &view, nil
}

func hydrateAgentMode(view *View) {
	if view == nil {
		return
	}
	if view.AgentMode == "" && view.Meta != nil {
		if raw, ok := view.Meta["agentMode"].(string); ok {
			view.AgentMode = strings.TrimSpace(raw)
		}
	}
	if mode, err := normalizeAgentMode(view.AgentMode); err == nil {
		view.AgentMode = mode
	} else {
		view.AgentMode = AgentModeCoding
	}
}

// sessionStreamURL returns the session-scoped SSE path (always, even for blank sessions).
func sessionStreamURL(sessionID string) string {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return ""
	}
	return "/api/v1/agents/sessions/" + sessionID + "/stream"
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// truncateTitle takes the first line and caps length by Unicode runes.
func truncateTitle(text string, maxRunes int) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	if i := strings.IndexByte(text, '\n'); i >= 0 {
		text = strings.TrimSpace(text[:i])
	}
	if maxRunes <= 0 {
		return text
	}
	runes := []rune(text)
	if len(runes) <= maxRunes {
		return string(runes)
	}
	return string(runes[:maxRunes])
}

func normalizeDisabledTools(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, name := range in {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func coerceStringSlice(raw any) []string {
	switch v := raw.(type) {
	case []string:
		return normalizeDisabledTools(v)
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return normalizeDisabledTools(out)
	default:
		return nil
	}
}

func (v *View) effectiveDisabledTools() []string {
	if v == nil {
		return nil
	}
	if len(v.DisabledTools) > 0 {
		return normalizeDisabledTools(v.DisabledTools)
	}
	if v.Meta != nil {
		return coerceStringSlice(v.Meta["disabledTools"])
	}
	return nil
}

// ToolDisabled reports whether a built-in tool is session-disabled.
func (v *View) ToolDisabled(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" || v == nil {
		return false
	}
	for _, d := range v.effectiveDisabledTools() {
		if d == name {
			return true
		}
	}
	return false
}

// DisabledToolsForRun returns disabled built-in tools for any session bound to runID.
func (s *Service) DisabledToolsForRun(runID string) []string {
	runID = strings.TrimSpace(runID)
	if s == nil || runID == "" {
		return nil
	}
	var rows []store.AuditLog
	if err := s.q().Where("event_type = ? AND run_id = ?", auditEventType, runID).Find(&rows).Error; err != nil {
		return nil
	}
	for _, row := range rows {
		view, err := decodeView(row)
		if err != nil || view == nil {
			continue
		}
		if tools := view.effectiveDisabledTools(); len(tools) > 0 {
			return tools
		}
	}
	return nil
}

// AllowedToolsSessionForRun returns session-scoped tool allows for any session bound to runID.
func (s *Service) AllowedToolsSessionForRun(runID string) []string {
	runID = strings.TrimSpace(runID)
	if s == nil || runID == "" {
		return nil
	}
	var rows []store.AuditLog
	if err := s.q().Where("event_type = ? AND run_id = ?", auditEventType, runID).Find(&rows).Error; err != nil {
		return nil
	}
	var out []string
	seen := map[string]bool{}
	for _, row := range rows {
		view, err := decodeView(row)
		if err != nil || view == nil || view.Meta == nil {
			continue
		}
		for _, tool := range coerceStringSlice(view.Meta[MetaAllowedToolsSession]) {
			if tool == "" || seen[tool] {
				continue
			}
			seen[tool] = true
			out = append(out, tool)
		}
	}
	return out
}

// AddAllowedToolSession appends tool to allowedToolsSession on sessions bound to runID.
func (s *Service) AddAllowedToolSession(runID, tool string) error {
	runID = strings.TrimSpace(runID)
	tool = strings.TrimSpace(tool)
	if s == nil || runID == "" || tool == "" {
		return nil
	}
	var rows []store.AuditLog
	if err := s.q().Where("event_type = ? AND run_id = ?", auditEventType, runID).Find(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		view, err := decodeView(row)
		if err != nil || view == nil {
			continue
		}
		if view.Meta == nil {
			view.Meta = map[string]any{}
		}
		list := coerceStringSlice(view.Meta[MetaAllowedToolsSession])
		exists := false
		for _, existing := range list {
			if existing == tool {
				exists = true
				break
			}
		}
		if !exists {
			view.Meta[MetaAllowedToolsSession] = append(list, tool)
			if err := s.save(view); err != nil {
				return err
			}
		}
	}
	return nil
}
