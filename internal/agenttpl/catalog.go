package agenttpl

import (
	"fmt"
	"sync"

	"github.com/ash-repwiki/ash/internal/plugins"
)

const (
	StatusApproved  = "approved"
	StatusCandidate = "candidate"
)

// Entry is a cataloged template with review status.
type Entry struct {
	Manifest Manifest `json:"manifest"`
	Status   string   `json:"status"`
}

// MarshalJSON flattens status next to manifest fields for list-friendly payloads.
// (API returns Entry as-is with nested manifest; approve wraps in {entry}.)

var (
	catalogMu sync.RWMutex
	catalog   = map[string]Entry{}
)

// SubmitCandidate validates and stores a custom template as candidate (B04).
func SubmitCandidate(m Manifest) (Entry, error) {
	loaded, err := Load(m)
	if err != nil {
		return Entry{}, err
	}
	if loaded.ID == "" {
		return Entry{}, fmt.Errorf("template id is required")
	}
	if isBuiltinID(loaded.ID) {
		return Entry{}, fmt.Errorf("cannot override built-in template %s", loaded.ID)
	}
	catalogMu.Lock()
	defer catalogMu.Unlock()
	if prev, ok := catalog[loaded.ID]; ok && prev.Status == StatusApproved {
		return Entry{}, fmt.Errorf("template %s is already approved", loaded.ID)
	}
	entry := Entry{Manifest: loaded, Status: StatusCandidate}
	catalog[loaded.ID] = entry
	return entry, nil
}

// ApproveTemplate marks a candidate template production-ready.
func ApproveTemplate(id string) (Entry, error) {
	catalogMu.Lock()
	defer catalogMu.Unlock()
	entry, ok := catalog[id]
	if !ok {
		return Entry{}, fmt.Errorf("unknown template %s", id)
	}
	entry.Status = StatusApproved
	catalog[id] = entry
	return entry, nil
}

// GetProduction returns a built-in or approved custom template.
// Candidate customs are rejected (H05).
func GetProduction(id string) (Manifest, error) {
	if m, err := Get(id); err == nil {
		return m, nil
	}
	catalogMu.RLock()
	entry, ok := catalog[id]
	catalogMu.RUnlock()
	if !ok {
		return Manifest{}, fmt.Errorf("unknown template %s", id)
	}
	if entry.Status != StatusApproved {
		return Manifest{}, fmt.Errorf("template %s is not approved for production", id)
	}
	return entry.Manifest, nil
}

// ListAll returns built-ins plus custom catalog entries.
func ListAll() (ListResponse, error) {
	base, err := List()
	if err != nil {
		return ListResponse{}, err
	}
	catalogMu.RLock()
	defer catalogMu.RUnlock()
	for _, entry := range catalog {
		base.Items = append(base.Items, entry.Manifest)
	}
	return base, nil
}

// ResetCatalogForTest clears custom templates (tests only).
func ResetCatalogForTest() {
	catalogMu.Lock()
	catalog = map[string]Entry{}
	catalogMu.Unlock()
}

func isBuiltinID(id string) bool {
	for _, b := range []Manifest{React(), PlanSolve(), Reviewer(), MemoryCurator(), Research()} {
		if b.ID == id {
			return true
		}
	}
	return false
}

// BindTemplateMemory checks memory components against the plugin host.
func BindTemplateMemory(m Manifest, host *plugins.Host) error {
	if host == nil {
		host = plugins.NewBuiltinHost()
	}
	if err := host.BindProduction(m.Memory); err != nil {
		return err
	}
	return host.BindProduction(m.PlanMemory)
}
