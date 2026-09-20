// Package ingress is the single-channel inbound seam (v6.4 VX41).
// Adapters: null and webhook-github. No IM SDKs.
package ingress

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

// RawInbound is one HTTP-ish inbound payload before channel-specific parsing.
type RawInbound struct {
	Channel string
	Headers map[string]string
	Body    []byte
}

// Event is a normalized ingress acceptance result (does not create runs by itself).
type Event struct {
	Channel    string            `json:"channel"`
	DeliveryID string            `json:"deliveryId,omitempty"`
	Kind       string            `json:"kind,omitempty"`
	Summary    string            `json:"summary,omitempty"`
	Meta       map[string]string `json:"meta,omitempty"`
}

// Adapter accepts inbound messages for one channel family.
type Adapter interface {
	Name() string
	Accept(ctx context.Context, in RawInbound) (*Event, error)
}

// Null discards all inbound messages.
type Null struct{}

func (Null) Name() string { return "null" }

func (Null) Accept(context.Context, RawInbound) (*Event, error) {
	return nil, fmt.Errorf("ingress null: discarded")
}

// WebhookGitHub normalizes GitHub webhook headers (HMAC stays in the HTTP handler).
type WebhookGitHub struct{}

func (WebhookGitHub) Name() string { return "webhook-github" }

func (WebhookGitHub) Accept(_ context.Context, in RawInbound) (*Event, error) {
	h := in.Headers
	if h == nil {
		h = map[string]string{}
	}
	delivery := headerGet(h, "X-GitHub-Delivery")
	event := headerGet(h, "X-GitHub-Event")
	if delivery == "" && event == "" && len(in.Body) == 0 {
		return nil, fmt.Errorf("webhook-github: empty delivery")
	}
	return &Event{
		Channel:    "webhook-github",
		DeliveryID: delivery,
		Kind:       event,
		Summary:    "github webhook accepted",
		Meta: map[string]string{
			"event":    event,
			"delivery": delivery,
		},
	}, nil
}

func headerGet(h map[string]string, key string) string {
	if v, ok := h[key]; ok {
		return strings.TrimSpace(v)
	}
	// case-insensitive fallback
	want := strings.ToLower(key)
	for k, v := range h {
		if strings.ToLower(k) == want {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// KnownAdapterIDs is the static catalog of ingress adapters ASH understands (VX41/VX42).
func KnownAdapterIDs() []string {
	return []string{"null", "webhook-github"}
}

// FromEnv selects a default adapter via ASH_INGRESS (null|webhook-github). Default null.
func FromEnv() Adapter {
	switch strings.ToLower(strings.TrimSpace(getenv("ASH_INGRESS"))) {
	case "webhook-github", "github", "webhook":
		return WebhookGitHub{}
	case "", "null", "none", "off", "no":
		return Null{}
	default:
		return Null{}
	}
}

// Recorder captures Accept results for tests.
type Recorder struct {
	mu   sync.Mutex
	name string
	evs  []Event
	err  error
}

func NewRecorder() *Recorder { return &Recorder{name: "recorder"} }

func (r *Recorder) Name() string {
	if r == nil || r.name == "" {
		return "recorder"
	}
	return r.name
}

func (r *Recorder) Accept(_ context.Context, in RawInbound) (*Event, error) {
	if r == nil {
		return nil, fmt.Errorf("nil recorder")
	}
	if r.err != nil {
		return nil, r.err
	}
	ev := Event{Channel: in.Channel, Summary: "recorded", Meta: map[string]string{}}
	r.mu.Lock()
	r.evs = append(r.evs, ev)
	r.mu.Unlock()
	return &ev, nil
}

func (r *Recorder) Events() []Event {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]Event, len(r.evs))
	copy(out, r.evs)
	return out
}

// getenv is overridden in tests when needed; production uses os.Getenv via stub below.
var getenv = func(k string) string {
	return stdGetenv(k)
}
