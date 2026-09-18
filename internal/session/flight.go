package session

import (
	"context"
	"sync"
)

// flight tracks an in-progress PromptTurn generation so Intent stop can cancel it.
type flight struct {
	cancel context.CancelFunc
	turnID string
	mu     sync.Mutex
	text   string // accumulated assistant text (blank-session live preview)
}

type flightRegistry struct {
	mu   sync.Mutex
	byID map[string]*flight
}

func newFlightRegistry() *flightRegistry {
	return &flightRegistry{byID: map[string]*flight{}}
}

func (r *flightRegistry) begin(sessionID string) (context.Context, *flight) {
	if r == nil || sessionID == "" {
		ctx, cancel := context.WithCancel(context.Background())
		return ctx, &flight{cancel: cancel}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if old, ok := r.byID[sessionID]; ok && old != nil && old.cancel != nil {
		old.cancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	f := &flight{cancel: cancel}
	r.byID[sessionID] = f
	return ctx, f
}

func (r *flightRegistry) end(sessionID string, f *flight) {
	if r == nil || sessionID == "" || f == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if cur, ok := r.byID[sessionID]; ok && cur == f {
		delete(r.byID, sessionID)
	}
	f.cancel()
}

// cancel stops an in-flight generation. Returns true if a flight was found.
func (r *flightRegistry) cancel(sessionID string) bool {
	if r == nil || sessionID == "" {
		return false
	}
	r.mu.Lock()
	f := r.byID[sessionID]
	r.mu.Unlock()
	if f == nil || f.cancel == nil {
		return false
	}
	f.cancel()
	return true
}

// has reports whether a PromptTurn flight is registered for the session.
func (r *flightRegistry) has(sessionID string) bool {
	if r == nil || sessionID == "" {
		return false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.byID[sessionID] != nil
}

func (f *flight) setPartial(turnID, text string) {
	if f == nil {
		return
	}
	f.mu.Lock()
	f.turnID = turnID
	f.text = text
	f.mu.Unlock()
}

func (f *flight) partial() (turnID, text string) {
	if f == nil {
		return "", ""
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.turnID, f.text
}

func (r *flightRegistry) partial(sessionID string) (turnID, text string, ok bool) {
	if r == nil || sessionID == "" {
		return "", "", false
	}
	r.mu.Lock()
	f := r.byID[sessionID]
	r.mu.Unlock()
	if f == nil {
		return "", "", false
	}
	turnID, text = f.partial()
	if text == "" {
		return turnID, text, false
	}
	return turnID, text, true
}
