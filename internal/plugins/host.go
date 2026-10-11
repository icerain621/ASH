// Package plugins is the in-process component host for agent and memory units.
//
// Process-out gRPC plugins (Indexer and similar) stay under internal/pluginabi
// with a separate naming surface (B03). Do not register gRPC ABI ids here.
package plugins

import (
	"fmt"
	"sort"
	"sync"
)

const (
	KindMemory      = "memory"
	KindTool        = "tool"
	KindLoopPolicy  = "loop_policy"
	KindCompaction  = "compaction"
	KindReviewer    = "reviewer"

	StatusApproved  = "approved"
	StatusCandidate = "candidate"

	MemRetrieve = "mem.retrieve"
	MemInject   = "mem.inject"
)

// Component is one replaceable agent or memory unit.
type Component struct {
	ID     string `json:"id"`
	Kind   string `json:"kind"`
	Status string `json:"status"`
}

// Host is the in-process registry shared by agent and memory components.
type Host struct {
	mu   sync.RWMutex
	byID map[string]Component
}

var (
	globalMu sync.Mutex
	global   *Host
)

// Global returns the process-wide host used by harness promote and HTTP.
func Global() *Host {
	globalMu.Lock()
	defer globalMu.Unlock()
	if global == nil {
		global = NewBuiltinHost()
	}
	return global
}

// ResetGlobalForTest replaces the process-wide host (tests only).
func ResetGlobalForTest() {
	globalMu.Lock()
	global = NewBuiltinHost()
	globalMu.Unlock()
}

// SetGlobalForTest installs h as the process-wide host (tests only).
func SetGlobalForTest(h *Host) {
	globalMu.Lock()
	global = h
	globalMu.Unlock()
}

// NewEmptyHost returns a host with no registered components.
func NewEmptyHost() *Host {
	return &Host{byID: map[string]Component{}}
}

// NewBuiltinHost registers the built-in memory and agent slot components as approved.
func NewBuiltinHost() *Host {
	h := &Host{byID: map[string]Component{}}
	for _, id := range []string{
		MemRetrieve,
		MemInject,
		"mem.propose",
		"mem.consolidate",
		"mem.forget",
		"mem.feedback",
		"mem.knowledge",
	} {
		h.byID[id] = Component{ID: id, Kind: KindMemory, Status: StatusApproved}
	}
	for _, c := range []Component{
		{ID: "tool.read", Kind: KindTool, Status: StatusApproved},
		{ID: "tool.write", Kind: KindTool, Status: StatusApproved},
		{ID: "loop.react", Kind: KindLoopPolicy, Status: StatusApproved},
		{ID: "loop.plan_solve", Kind: KindLoopPolicy, Status: StatusApproved},
		{ID: "compaction.threshold", Kind: KindCompaction, Status: StatusApproved},
		{ID: "reviewer.default", Kind: KindReviewer, Status: StatusApproved},
	} {
		h.byID[c.ID] = c
	}
	return h
}

// Register adds or replaces a component as candidate.
// Production status is only granted via Approve (or NewBuiltinHost seeds).
func (h *Host) Register(c Component) error {
	if c.ID == "" {
		return fmt.Errorf("component id is required")
	}
	if c.Kind == "" {
		c.Kind = KindMemory
	}
	if !validKind(c.Kind) {
		return fmt.Errorf("unknown component kind %q", c.Kind)
	}
	c.Status = StatusCandidate
	h.mu.Lock()
	defer h.mu.Unlock()
	if prev, ok := h.byID[c.ID]; ok && prev.Status == StatusApproved {
		return fmt.Errorf("component %s is already approved; replace via Approve after review", c.ID)
	}
	h.byID[c.ID] = c
	return nil
}

// Approve marks a candidate component production-ready.
func (h *Host) Approve(id string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	c, ok := h.byID[id]
	if !ok {
		return fmt.Errorf("component %s not found", id)
	}
	c.Status = StatusApproved
	h.byID[id] = c
	return nil
}

// List returns registered components sorted by id.
func (h *Host) List() []Component {
	h.mu.RLock()
	defer h.mu.RUnlock()
	out := make([]Component, 0, len(h.byID))
	for _, c := range h.byID {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// Get returns one component.
func (h *Host) Get(id string) (Component, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	c, ok := h.byID[id]
	return c, ok
}

// GetProduction returns an approved component or a typed bind error (D08).
func (h *Host) GetProduction(id string) (Component, error) {
	c, ok := h.Get(id)
	if !ok {
		return Component{}, fmt.Errorf("%w: %s", ErrUnknown, id)
	}
	if c.Status != StatusApproved {
		return Component{}, fmt.Errorf("%w: %s", ErrNotApproved, id)
	}
	return c, nil
}

// BindProduction rejects unknown, candidate, or illegal memory combinations.
func (h *Host) BindProduction(ids []string) error {
	seen := map[string]struct{}{}
	for _, id := range ids {
		if _, err := h.GetProduction(id); err != nil {
			return err
		}
		seen[id] = struct{}{}
	}
	if _, inject := seen[MemInject]; inject {
		if _, retrieve := seen[MemRetrieve]; !retrieve {
			return fmt.Errorf("%w: mem.inject requires mem.retrieve", ErrInvalidBind)
		}
	}
	return nil
}

func validKind(k string) bool {
	switch k {
	case KindMemory, KindTool, KindLoopPolicy, KindCompaction, KindReviewer:
		return true
	default:
		return false
	}
}
