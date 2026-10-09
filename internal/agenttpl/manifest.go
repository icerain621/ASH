package agenttpl

import (
	"fmt"

	"github.com/ash-repwiki/ash/internal/plugins"
)

const Version = "1.0.0"

// ContractTools are the names a template may declare.
func ContractTools() []string {
	return []string{"read", "write", "edit", "bash", "rag.query"}
}

// Manifest is the component list for one agent template.
type Manifest struct {
	ID         string   `json:"id"`
	Version    string   `json:"version"`
	Loop       string   `json:"loop"`
	Tools      []string `json:"tools"`
	Compaction string   `json:"compaction"`
	Sandbox    string   `json:"sandbox"`
	Skills     []string `json:"skills"`
	Hooks      []string `json:"hooks"`
	MaxTurns   int      `json:"maxTurns"`
	Memory     []string `json:"memory"`
	PlanMemory []string `json:"planMemory,omitempty"`
}

// ListResponse is the catalog payload.
type ListResponse struct {
	Items []Manifest `json:"items"`
}

// Load rejects illegal memory combinations, unknown tools, and unbounded loops.
func Load(m Manifest) (Manifest, error) {
	if m.MaxTurns < 1 {
		return Manifest{}, fmt.Errorf("template %s maxTurns must be >= 1", m.ID)
	}
	if err := validateTools(m.Tools); err != nil {
		return Manifest{}, fmt.Errorf("template %s: %w", m.ID, err)
	}
	host := plugins.NewBuiltinHost()
	if err := host.BindProduction(m.Memory); err != nil {
		return Manifest{}, fmt.Errorf("template %s: %w", m.ID, err)
	}
	if err := host.BindProduction(m.PlanMemory); err != nil {
		return Manifest{}, fmt.Errorf("template %s plan memory: %w", m.ID, err)
	}
	return m, nil
}

// List returns the five built-in templates.
func List() (ListResponse, error) {
	raw := []Manifest{React(), PlanSolve(), Reviewer(), MemoryCurator(), Research()}
	items := make([]Manifest, 0, len(raw))
	for _, item := range raw {
		loaded, err := Load(item)
		if err != nil {
			return ListResponse{}, err
		}
		items = append(items, loaded)
	}
	return ListResponse{Items: items}, nil
}

// Get loads one built-in template by id.
func Get(id string) (Manifest, error) {
	list, err := List()
	if err != nil {
		return Manifest{}, err
	}
	for _, item := range list.Items {
		if item.ID == id {
			return item, nil
		}
	}
	return Manifest{}, fmt.Errorf("unknown template %s", id)
}

func validateTools(names []string) error {
	allowed := map[string]struct{}{}
	for _, name := range ContractTools() {
		allowed[name] = struct{}{}
	}
	for _, name := range names {
		if _, ok := allowed[name]; !ok {
			return fmt.Errorf("tool %s is outside the contract", name)
		}
	}
	return nil
}
