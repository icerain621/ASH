package review

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/ash-repwiki/ash/internal/memory"
)

// Kind is one reviewable object class (F03).
type Kind string

const (
	KindAgentTemplate  Kind = "agent_template"
	KindHarnessProfile Kind = "harness_profile"
	KindMemoryRecord   Kind = "memory_record"
	KindArtifact       Kind = "artifact"
	KindRunSample      Kind = "run_sample"
)

// Status of a queue item.
const (
	StatusPending  = "pending"
	StatusApproved = "approved"
	StatusRejected = "rejected"
)

// Item is one entry on the review queue.
type Item struct {
	ID       string `json:"id"`
	Kind     Kind   `json:"kind"`
	TargetID string `json:"targetId"`
	Status   string `json:"status"`
	Reason   string `json:"reason,omitempty"`
	Created  int64  `json:"created"`
	Decided  int64  `json:"decided,omitempty"`
}

// Service is an in-process review queue for P4 (tables/RLS land in P5 G01).
type Service struct {
	mu    sync.Mutex
	items map[string]Item
}

// NewService creates an empty queue.
func NewService() *Service {
	return &Service{items: map[string]Item{}}
}

// Submit enqueues a reviewable target.
func (s *Service) Submit(kind Kind, targetID, reason string) (Item, error) {
	if !validKind(kind) {
		return Item{}, fmt.Errorf("unknown review kind %q", kind)
	}
	targetID = strings.TrimSpace(targetID)
	if targetID == "" {
		return Item{}, fmt.Errorf("targetId is required")
	}
	item := Item{
		ID: "rev_" + uuid.NewString(), Kind: kind, TargetID: targetID,
		Status: StatusPending, Reason: strings.TrimSpace(reason),
		Created: time.Now().UTC().Unix(),
	}
	s.mu.Lock()
	s.items[item.ID] = item
	s.mu.Unlock()
	return item, nil
}

// Decide approves or rejects a pending item.
func (s *Service) Decide(id, decision, reason string) (Item, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.items[id]
	if !ok {
		return Item{}, fmt.Errorf("review item %s not found", id)
	}
	if item.Status != StatusPending {
		return Item{}, fmt.Errorf("review item %s status %q", id, item.Status)
	}
	switch strings.ToLower(strings.TrimSpace(decision)) {
	case "approve":
		item.Status = StatusApproved
	case "reject":
		item.Status = StatusRejected
	default:
		return Item{}, fmt.Errorf("decision must be approve|reject")
	}
	if r := strings.TrimSpace(reason); r != "" {
		item.Reason = r
	}
	item.Decided = time.Now().UTC().Unix()
	s.items[id] = item
	return item, nil
}

// List returns pending items, optionally filtered by kind.
func (s *Service) List(kind Kind) []Item {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Item, 0, len(s.items))
	for _, item := range s.items {
		if item.Status != StatusPending {
			continue
		}
		if kind != "" && item.Kind != kind {
			continue
		}
		out = append(out, item)
	}
	return out
}

// EvalCounts maps feedback kinds onto the three memory evaluation counters (F05).
type EvalCounts struct {
	Remembered    int `json:"remembered"`
	UsedCorrectly int `json:"usedCorrectly"`
	WrongUpdate   int `json:"wrongUpdate"`
}

// MemoryEvalCounts tallies hit_used as remembered plus the two correctness kinds.
func MemoryEvalCounts(kinds []string) EvalCounts {
	fb := memory.TallyFeedback(kinds)
	return EvalCounts{
		Remembered:    fb.HitUsed,
		UsedCorrectly: fb.UsedCorrectly,
		WrongUpdate:   fb.WrongUpdate,
	}
}

// QualityCaseIDs lists Doctor cases that are quality (not boot) under F02 transition.
func QualityCaseIDs() []string {
	return []string{
		"TR0-01", "TR0-04", "TR0-06",
		"TR1-02", "TR1-03", "TR1-06",
		"TR3-04",
	}
}

func validKind(k Kind) bool {
	switch k {
	case KindAgentTemplate, KindHarnessProfile, KindMemoryRecord, KindArtifact, KindRunSample:
		return true
	default:
		return false
	}
}
