package session

import (
	"errors"
	"fmt"
	"strings"

	"github.com/ash-repwiki/ash/internal/events"
	"github.com/ash-repwiki/ash/internal/interaction"
)

// Intent actions for thin agent interaction (GV01 / EW04).
const (
	IntentPrompt  = "prompt"
	IntentApprove = "approve"
	IntentCancel  = "cancel"
	IntentReject  = "reject"
	IntentCommand = "command"

	// Gate aliases (sprint EW04 protocol names).
	IntentAllowOnce    = "allow_once"
	IntentAllowSession = "allow_session"
	IntentDeny         = "deny"
	IntentCancelRun    = "cancel_run"

	ApproveScopeOnce    = "once"
	ApproveScopeSession = "session"

	MetaAllowedToolsSession = "allowedToolsSession"
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
		if scope == ApproveScopeSession && tool != "" {
			if err := s.addAllowedToolSession(view, tool); err != nil {
				return nil, fmt.Errorf("%w: %v", ErrIntentRejected, err)
			}
		}
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

func (s *Service) addAllowedToolSession(view *View, tool string) error {
	if view == nil || strings.TrimSpace(tool) == "" {
		return nil
	}
	if view.Meta == nil {
		view.Meta = map[string]any{}
	}
	list := coerceStringSlice(view.Meta[MetaAllowedToolsSession])
	tool = strings.TrimSpace(tool)
	for _, existing := range list {
		if existing == tool {
			return s.save(view)
		}
	}
	view.Meta[MetaAllowedToolsSession] = append(list, tool)
	return s.save(view)
}

// intentCancel cancels the bound run when present, and/or cancels an in-flight PromptTurn
// (blank or bound) so LLM/provider generation can stop with assistant.message stopped:true.
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
