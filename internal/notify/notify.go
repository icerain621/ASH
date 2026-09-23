// Package notify is the delivery seam (v6.3 VX33; webhook ingest VX61).
// Adapters: null (default) and log. No IM SDKs.
package notify

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
)

// Event is one delivery payload (duty completion, future Cron, etc.).
type Event struct {
	Kind    string            `json:"kind"`
	Title   string            `json:"title"`
	Body    string            `json:"body,omitempty"`
	SpaceID string            `json:"spaceId,omitempty"`
	Meta    map[string]string `json:"meta,omitempty"`
}

// Notifier delivers Events. Implementations must be safe for concurrent use.
type Notifier interface {
	Name() string
	Notify(ctx context.Context, ev Event) error
}

// Null discards all events.
type Null struct{}

func (Null) Name() string { return "null" }

func (Null) Notify(context.Context, Event) error { return nil }

// Log writes events via the standard logger.
type Log struct {
	Logger *log.Logger
}

func (l Log) Name() string { return "log" }

func (l Log) Notify(_ context.Context, ev Event) error {
	lg := l.Logger
	if lg == nil {
		lg = log.Default()
	}
	lg.Printf("ash.notify kind=%s title=%q space=%s body=%q meta=%v",
		ev.Kind, ev.Title, ev.SpaceID, ev.Body, ev.Meta)
	return nil
}

// Recorder captures events for tests.
type Recorder struct {
	mu   sync.Mutex
	evs  []Event
	name string
}

func NewRecorder() *Recorder { return &Recorder{name: "recorder"} }

func (r *Recorder) Name() string {
	if r == nil || r.name == "" {
		return "recorder"
	}
	return r.name
}

func (r *Recorder) Notify(_ context.Context, ev Event) error {
	if r == nil {
		return fmt.Errorf("nil recorder")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := ev
	if ev.Meta != nil {
		cp.Meta = make(map[string]string, len(ev.Meta))
		for k, v := range ev.Meta {
			cp.Meta[k] = v
		}
	}
	r.evs = append(r.evs, cp)
	return nil
}

func (r *Recorder) Events() []Event {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]Event, len(r.evs))
	for i, ev := range r.evs {
		out[i] = ev
		if ev.Meta != nil {
			out[i].Meta = make(map[string]string, len(ev.Meta))
			for k, v := range ev.Meta {
				out[i].Meta[k] = v
			}
		}
	}
	return out
}

// FromEnv selects notifier via ASH_NOTIFIER (null|log). Default and unknown → null.
func FromEnv() Notifier {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("ASH_NOTIFIER"))) {
	case "log":
		return Log{}
	case "", "null", "none", "off", "no":
		return Null{}
	default:
		return Null{}
	}
}
