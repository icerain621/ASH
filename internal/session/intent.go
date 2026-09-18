package session

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ash-repwiki/ash/internal/events"
	"github.com/ash-repwiki/ash/internal/interaction"
	"github.com/ash-repwiki/ash/internal/store"
)

// Intent actions for thin agent interaction (GV01 / EW04).
const (
	IntentPrompt  = "prompt"
	IntentApprove = "approve"
	IntentCancel  = "cancel"
	IntentReject  = "reject"
	IntentCommand = "command"
	IntentSteer   = "steer"
	IntentQueue   = "queue"

	// MetaFollowUpQueue is the ordered list of prompts waiting until the current
	// turn or bound run finishes. steer and queue are mutually exclusive:
	// steer cancels in-flight work then prompts immediately; queue never cancels.
	// cancel/stop stops only the current turn/run and does not clear this list,
	// and a canceled turn/run does not drain it.
	MetaFollowUpQueue = "followUpQueue"

	// Gate aliases (sprint EW04 protocol names).
	IntentAllowOnce    = "allow_once"
	IntentAllowSession = "allow_session"
	IntentDeny         = "deny"
	IntentCancelRun    = "cancel_run"

	ApproveScopeOnce    = "once"
	ApproveScopeSession = "session"

	MetaAllowedToolsSession = "allowedToolsSession"

	// How long steer waits for a canceled in-flight PromptTurn to unwind before starting the new prompt.
	steerFlightWait = 3 * time.Second
)

// ErrIntentRejected is returned when an intent cannot be applied (fail-closed).
var ErrIntentRejected = errors.New("session intent rejected")

// GateApproveRequest is the run-gate approve payload from session intents.
type GateApproveRequest struct {
	ActorID string
	Reason  string
	Scope   string
	Tool    string
}

// RunControl abstracts run gate operations to avoid importing internal/runs (cycle).
type RunControl interface {
	ApproveRun(runID string, req GateApproveRequest) error
	CancelRun(runID string) error
}

// IntentRequest is the thin UI intent payload.
type IntentRequest struct {
	Action  string `json:"action" binding:"required"`
	Prompt  string `json:"prompt,omitempty"`
	Reason  string `json:"reason,omitempty"`
	ActorID string `json:"actorId,omitempty"`
	Command string `json:"command,omitempty"`
	Args    string `json:"args,omitempty"`
	Scope   string `json:"scope,omitempty"`
	Tool    string `json:"tool,omitempty"`
}

// WithRunControl returns a shallow copy with run gate control wired.
func (s *Service) WithRunControl(rc RunControl) *Service {
	if s == nil {
		return nil
	}
	out := *s
	out.runs = rc
	return &out
}

// Intent applies a thin interaction intent against the session.
func (s *Service) Intent(sessionID string, req IntentRequest) (*View, error) {
	action := strings.ToLower(strings.TrimSpace(req.Action))
	switch action {
	case IntentPrompt:
		view, _, err := s.PromptTurn(sessionID, TurnRequest{Prompt: req.Prompt})
		return view, err
	case IntentSteer:
		return s.intentSteer(sessionID, req)
	case IntentQueue:
		return s.intentQueue(sessionID, req)
	case IntentApprove, IntentAllowOnce, IntentAllowSession:
		return s.intentApprove(sessionID, req, action)
	case IntentReject, IntentDeny:
		return s.intentApprove(sessionID, req, IntentReject)
	case IntentCancel, "stop", IntentCancelRun: // stop / cancel_run are UX aliases of cancel
		return s.intentCancel(sessionID, req)
	case IntentCommand:
		return s.intentCommand(sessionID, req)
	default:
		return nil, fmt.Errorf("%w: unknown action %q", ErrIntentRejected, req.Action)
	}
}

// intentSteer cancels in-flight generation and/or an active bound run, then applies a new prompt.
// Requires active work (in-flight PromptTurn or non-terminal bound run); idle sessions are rejected
// (steer is not a prompt alias — use action=prompt when idle).
// Mutually exclusive with queue: steer interrupts now; queue never cancels and waits in
// meta.followUpQueue. steer does not delete that list.
func (s *Service) intentSteer(sessionID string, req IntentRequest) (*View, error) {
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		return nil, fmt.Errorf("%w: steer requires prompt", ErrIntentRejected)
	}
	view, err := s.Get(sessionID)
	if err != nil {
		return nil, err
	}
	if view.Status != StatusActive {
		return nil, fmt.Errorf("%w: session status %q", ErrIntentRejected, view.Status)
	}

	cancelledFlight := false
	if s.flights != nil {
		cancelledFlight = s.flights.cancel(sessionID)
	}
	activeRun := s.boundRunIsActive(view.RunID)
	if !cancelledFlight && !activeRun {
		return nil, fmt.Errorf("%w: nothing to steer (no in-flight turn or active run)", ErrIntentRejected)
	}

	if activeRun {
		if s.runs == nil {
			return nil, fmt.Errorf("%w: run control is not configured", ErrIntentRejected)
		}
		if err := s.runs.CancelRun(view.RunID); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrIntentRejected, err)
		}
	}

	if cancelledFlight {
		deadline := time.Now().Add(steerFlightWait)
		for time.Now().Before(deadline) && s.flights != nil && s.flights.has(sessionID) {
			time.Sleep(10 * time.Millisecond)
		}
	}

	actor := firstNonEmpty(strings.TrimSpace(req.ActorID), view.CreatedBy, "session")
	if view.RunID != "" && s.events != nil {
		trace := firstNonEmpty(view.TraceID, view.RunID)
		_, _ = s.events.Append(view.RunID, trace, "session.steer", "info", map[string]any{
			"sessionId": view.ID, "action": IntentSteer, "actorId": actor, "prompt": prompt,
			"canceledFlight": cancelledFlight, "canceledRun": activeRun,
			"threadId": metaString(view.Meta, interaction.MetaThreadID),
		}, events.WithVisibility(events.VisibilityUIOnly))
	}

	view, _, err = s.PromptTurn(sessionID, TurnRequest{Prompt: prompt})
	return view, err
}

// intentQueue appends a follow-up prompt without interrupting in-flight work.
// Unlike steer, queue never cancels. Busy (in-flight PromptTurn or non-terminal
// bound run) appends meta.followUpQueue. Idle is treated as prompt.
// One queued item drains after a successful PromptTurn, or after the bound run
// finishes/fails. cancel/stop does not clear or drain the list.
func (s *Service) intentQueue(sessionID string, req IntentRequest) (*View, error) {
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		return nil, fmt.Errorf("%w: queue requires prompt", ErrIntentRejected)
	}
	view, err := s.Get(sessionID)
	if err != nil {
		return nil, err
	}
	if view.Status != StatusActive {
		return nil, fmt.Errorf("%w: session status %q", ErrIntentRejected, view.Status)
	}
	if !s.sessionBusy(sessionID, view) {
		out, _, err := s.PromptTurn(sessionID, TurnRequest{Prompt: prompt})
		return out, err
	}
	return s.enqueueFollowUp(sessionID, prompt, req.ActorID)
}

func (s *Service) sessionBusy(sessionID string, view *View) bool {
	if s != nil && s.flights != nil && s.flights.has(sessionID) {
		return true
	}
	if view == nil {
		return false
	}
	return s.boundRunIsActive(view.RunID)
}

func (s *Service) lockSessionDoc() func() {
	if s == nil || s.flights == nil {
		return func() {}
	}
	return s.flights.lockDoc()
}

func (s *Service) enqueueFollowUp(sessionID, prompt, actorID string) (*View, error) {
	unlock := s.lockSessionDoc()
	view, err := s.Get(sessionID)
	if err != nil {
		unlock()
		return nil, err
	}
	if view.Status != StatusActive {
		unlock()
		return nil, fmt.Errorf("%w: session status %q", ErrIntentRejected, view.Status)
	}
	if !s.sessionBusy(sessionID, view) {
		unlock()
		out, _, err := s.PromptTurn(sessionID, TurnRequest{Prompt: prompt})
		return out, err
	}
	q := append(readFollowUpQueue(view.Meta), prompt)
	view.Meta = writeFollowUpQueue(view.Meta, q)
	view.UpdatedAt = time.Now().UTC().Unix()
	if err := s.save(view); err != nil {
		unlock()
		return nil, err
	}
	s.emitQueueEvent(view, "enqueue", prompt, len(q), actorID)
	unlock()
	view.StreamURL = sessionStreamURL(view.ID)
	return view, nil
}

// saveMergingFollowUpQueue persists view but keeps meta.followUpQueue from the
// latest stored document so a concurrent queue intent is not overwritten by a
// PromptTurn that loaded its view before the enqueue.
func (s *Service) saveMergingFollowUpQueue(view *View) error {
	unlock := s.lockSessionDoc()
	defer unlock()
	s.adoptLatestFollowUpQueue(view)
	return s.save(view)
}

func (s *Service) adoptLatestFollowUpQueue(view *View) {
	if view == nil {
		return
	}
	latest, err := s.Get(view.ID)
	if err != nil || latest == nil {
		return
	}
	view.Meta = writeFollowUpQueue(view.Meta, readFollowUpQueue(latest.Meta))
}

// drainOneFollowUp pops one queued prompt and runs it when the session is idle.
// No-op while a flight or non-terminal bound run is still active.
func (s *Service) drainOneFollowUp(sessionID string) {
	if s == nil || strings.TrimSpace(sessionID) == "" {
		return
	}
	unlock := s.lockSessionDoc()
	view, err := s.Get(sessionID)
	if err != nil || view == nil || view.Status != StatusActive || s.sessionBusy(sessionID, view) {
		unlock()
		return
	}
	q := readFollowUpQueue(view.Meta)
	if len(q) == 0 {
		unlock()
		return
	}
	next := q[0]
	view.Meta = writeFollowUpQueue(view.Meta, q[1:])
	view.UpdatedAt = time.Now().UTC().Unix()
	if err := s.save(view); err != nil {
		unlock()
		return
	}
	s.emitQueueEvent(view, "drain", next, len(q)-1, "")
	unlock()
	if _, _, err := s.PromptTurn(sessionID, TurnRequest{Prompt: next}); err != nil {
		s.requeueFront(sessionID, next)
	}
}

// DrainFollowUpForRun drains one follow-up on sessions bound to runID after the
// run finishes or fails. Canceled runs are ignored so stop does not start the queue.
func (s *Service) DrainFollowUpForRun(runID string) {
	runID = strings.TrimSpace(runID)
	if s == nil || s.db == nil || runID == "" || !s.runSettledForDrain(runID) {
		return
	}
	var rows []store.AuditLog
	if err := s.q().Where("event_type = ? AND run_id = ?", auditEventType, runID).Find(&rows).Error; err != nil {
		return
	}
	for _, row := range rows {
		s.drainOneFollowUp(row.ID)
	}
}

func (s *Service) runSettledForDrain(runID string) bool {
	var rec store.RunRecord
	if err := s.q().First(&rec, "id = ?", runID).Error; err != nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(rec.Status)) {
	case "finished", "failed":
		return true
	default:
		return false
	}
}

func (s *Service) requeueFront(sessionID, prompt string) {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return
	}
	unlock := s.lockSessionDoc()
	defer unlock()
	view, err := s.Get(sessionID)
	if err != nil || view == nil {
		return
	}
	q := append([]string{prompt}, readFollowUpQueue(view.Meta)...)
	view.Meta = writeFollowUpQueue(view.Meta, q)
	view.UpdatedAt = time.Now().UTC().Unix()
	_ = s.save(view)
}

func (s *Service) emitQueueEvent(view *View, phase, prompt string, queueLen int, actorID string) {
	if view == nil || strings.TrimSpace(view.RunID) == "" || s == nil || s.events == nil {
		return
	}
	actor := firstNonEmpty(strings.TrimSpace(actorID), view.CreatedBy, "session")
	trace := firstNonEmpty(view.TraceID, view.RunID)
	_, _ = s.events.Append(view.RunID, trace, "session.queue", "info", map[string]any{
		"sessionId": view.ID, "action": IntentQueue, "phase": phase, "actorId": actor,
		"prompt": prompt, "queueLen": queueLen,
		"threadId": metaString(view.Meta, interaction.MetaThreadID),
	}, events.WithVisibility(events.VisibilityUIOnly))
}

func readFollowUpQueue(meta map[string]any) []string {
	if meta == nil {
		return nil
	}
	raw, ok := meta[MetaFollowUpQueue]
	if !ok || raw == nil {
		return nil
	}
	var items []string
	switch v := raw.(type) {
	case []string:
		items = v
	case []any:
		items = make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				items = append(items, s)
			}
		}
	default:
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item != "" {
			out = append(out, item)
		}
	}
	return out
}

func writeFollowUpQueue(meta map[string]any, items []string) map[string]any {
	if meta == nil {
		meta = map[string]any{}
	}
	if len(items) == 0 {
		delete(meta, MetaFollowUpQueue)
		return meta
	}
	copied := append([]string(nil), items...)
	meta[MetaFollowUpQueue] = copied
	return meta
}

func (s *Service) boundRunIsActive(runID string) bool {
	runID = strings.TrimSpace(runID)
	if runID == "" || s == nil || s.db == nil {
		return false
	}
	var rec store.RunRecord
	if err := s.q().First(&rec, "id = ?", runID).Error; err != nil {
		return false
	}
	return !runStatusTerminal(rec.Status)
}

func runStatusTerminal(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "finished", "failed", "canceled", "cancelled":
		return true
	default:
		return false
	}
}

func (s *Service) intentApprove(sessionID string, req IntentRequest, action string) (*View, error) {
	view, err := s.Get(sessionID)
	if err != nil {
		return nil, err
	}
	if view.Status != StatusActive {
		return nil, fmt.Errorf("%w: session status %q", ErrIntentRejected, view.Status)
	}
	if strings.TrimSpace(view.RunID) == "" {
		return nil, fmt.Errorf("%w: session has no bound run", ErrIntentRejected)
	}
	if s.runs == nil {
		return nil, fmt.Errorf("%w: run control is not configured", ErrIntentRejected)
	}
	actor := firstNonEmpty(strings.TrimSpace(req.ActorID), view.CreatedBy, "session")
	reason := strings.TrimSpace(req.Reason)
	scope := normalizeIntentApproveScope(action, req.Scope)
	tool := strings.TrimSpace(req.Tool)

	if action == IntentReject {
		if reason == "" {
			reason = "rejected"
		}
		// Reject = cancel gate path for GV01 (evolve reject can thicken later).
		if err := s.runs.CancelRun(view.RunID); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrIntentRejected, err)
		}
	} else {
		if err := s.runs.ApproveRun(view.RunID, GateApproveRequest{
			ActorID: actor, Reason: reason, Scope: scope, Tool: tool,
		}); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrIntentRejected, err)
		}
		// Session allow-list widening is owned by runs.Approve (evidence tool only).
	}
	if view.RunID != "" && s.events != nil {
		trace := firstNonEmpty(view.TraceID, view.RunID)
		_, _ = s.events.Append(view.RunID, trace, "session.intent", "info", map[string]any{
			"sessionId": view.ID, "action": action, "actorId": actor, "reason": reason,
			"scope": scope, "tool": tool,
			"threadId": metaString(view.Meta, interaction.MetaThreadID),
		}, events.WithVisibility(events.VisibilityUIOnly))
	}
	view, err = s.Get(sessionID)
	if err != nil {
		return nil, err
	}
	return view, nil
}

func normalizeIntentApproveScope(action, scope string) string {
	action = strings.ToLower(strings.TrimSpace(action))
	if action == IntentAllowSession {
		return ApproveScopeSession
	}
	if action == IntentAllowOnce {
		return ApproveScopeOnce
	}
	switch strings.ToLower(strings.TrimSpace(scope)) {
	case ApproveScopeSession, IntentAllowSession:
		return ApproveScopeSession
	default:
		return ApproveScopeOnce
	}
}

// intentCancel cancels the bound run when present, and/or cancels an in-flight PromptTurn
// (blank or bound) so LLM/provider generation can stop with assistant.message stopped:true.
// Stop/cancel does not clear meta.followUpQueue and does not drain it. A canceled PromptTurn
// skips the post-turn drain; runs.Cancel does not notify the follow-up drainer.
func (s *Service) intentCancel(sessionID string, req IntentRequest) (*View, error) {
	view, err := s.Get(sessionID)
	if err != nil {
		return nil, err
	}
	cancelledFlight := false
	if s.flights != nil {
		cancelledFlight = s.flights.cancel(sessionID)
	}
	hasRun := strings.TrimSpace(view.RunID) != ""
	if hasRun {
		if s.runs == nil {
			return nil, fmt.Errorf("%w: run control is not configured", ErrIntentRejected)
		}
		if err := s.runs.CancelRun(view.RunID); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrIntentRejected, err)
		}
		actor := firstNonEmpty(strings.TrimSpace(req.ActorID), view.CreatedBy, "session")
		if s.events != nil {
			trace := firstNonEmpty(view.TraceID, view.RunID)
			_, _ = s.events.Append(view.RunID, trace, "session.intent", "info", map[string]any{
				"sessionId": view.ID, "action": IntentCancel, "actorId": actor,
				"threadId": metaString(view.Meta, interaction.MetaThreadID),
			}, events.WithVisibility(events.VisibilityUIOnly))
		}
	} else if !cancelledFlight {
		return nil, fmt.Errorf("%w: nothing to stop (no bound run or in-flight turn)", ErrIntentRejected)
	}
	view, err = s.Get(sessionID)
	if err != nil {
		return nil, err
	}
	view.StreamURL = sessionStreamURL(view.ID)
	return view, nil
}

func (s *Service) ensureMainThread(view *View) {
	if view == nil {
		return
	}
	tid, meta, created := interaction.EnsureMainThread(view.Meta, view.ID, view.RunID)
	view.Meta = meta
	if !created || view.RunID == "" || s.events == nil {
		return
	}
	trace := firstNonEmpty(view.TraceID, view.RunID)
	_, _ = s.events.Append(view.RunID, trace, "session.thread.ensured", "info", map[string]any{
		"sessionId": view.ID, "threadId": tid, "threadKind": interaction.ThreadKindMain,
	}, events.WithVisibility(events.VisibilityUIOnly))
}

func metaString(meta map[string]any, key string) string {
	if meta == nil {
		return ""
	}
	if v, ok := meta[key].(string); ok {
		return v
	}
	return ""
}
