package rules

// Deprecation records one legacy step kind pointed at a template replacement.
type Deprecation struct {
	StepID     string
	FromKind   string
	TemplateID string
}

// AdaptLegacy rewrites llm steps onto tpl.plan-solve and records deprecations for
// tool_chain / verify. Those two kinds still execute on the legacy path for one
// major version so tool gates, hooks, and sandbox floors keep working.
func AdaptLegacy(doc *Document) []Deprecation {
	if doc == nil {
		return nil
	}
	var out []Deprecation
	for i := range doc.Scenario.Steps {
		st := &doc.Scenario.Steps[i]
		switch st.Kind {
		case "llm":
			out = append(out, Deprecation{StepID: st.ID, FromKind: "llm", TemplateID: "tpl.plan-solve"})
			st.Kind = "agent"
			ensureAgent(st).TemplateID = "tpl.plan-solve"
			if st.Agent.Prompt == "" && st.PromptRef != "" {
				st.Agent.Prompt = st.PromptRef
			}
		case "tool_chain":
			out = append(out, Deprecation{StepID: st.ID, FromKind: "tool_chain", TemplateID: "tpl.react"})
		case "verify":
			out = append(out, Deprecation{StepID: st.ID, FromKind: "verify", TemplateID: "tpl.reviewer"})
		}
	}
	return out
}

func ensureAgent(st *Step) *AgentSpec {
	if st.Agent == nil {
		st.Agent = &AgentSpec{}
	}
	return st.Agent
}
