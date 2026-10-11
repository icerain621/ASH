package runs

import (
	"strings"

	"github.com/ash-repwiki/ash/internal/agenttpl"
	"github.com/ash-repwiki/ash/internal/plugins"
	"github.com/ash-repwiki/ash/internal/rules"
)

// collectProductionTemplateIDs returns distinct template ids that Create will bind.
func collectProductionTemplateIDs(doc *rules.Document) []string {
	if doc == nil {
		return nil
	}
	seen := map[string]struct{}{}
	var out []string
	for _, st := range doc.Scenario.Steps {
		id := ""
		if st.Agent != nil {
			id = strings.TrimSpace(st.Agent.TemplateID)
		}
		if id == "" && st.Kind == "review" {
			id = "tpl.reviewer"
		}
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

// validateProductionTemplates ensures every scenario template id and its memory
// components are production-bindable (G03/H05/A09/D08).
func validateProductionTemplates(doc *rules.Document) error {
	for _, id := range collectProductionTemplateIDs(doc) {
		manifest, err := agenttpl.GetProduction(id)
		if err != nil {
			return err
		}
		if err := agenttpl.BindTemplateMemory(manifest, plugins.Global()); err != nil {
			return err
		}
	}
	return nil
}
