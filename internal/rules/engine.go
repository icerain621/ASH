package rules

import (
	"fmt"
)

// Engine interprets validated DSL documents at runtime (gates/hooks).
type Engine struct {
	doc *Document
}

func NewEngine(doc *Document) *Engine {
	return &Engine{doc: doc}
}

// Document returns the bound scenario document.
func (e *Engine) Document() *Document {
	return e.doc
}

// RequiredInputs returns input keys that must be present in run inputs.
func (e *Engine) RequiredInputs() []string {
	if e.doc == nil || e.doc.Scenario.Inputs == nil {
		return nil
	}
	return append([]string(nil), e.doc.Scenario.Inputs.Required...)
}

// RequiredArtifactTypes returns artifact types required before finish.
func (e *Engine) RequiredArtifactTypes() []string {
	if e.doc == nil || e.doc.Scenario.Artifacts == nil {
		return nil
	}
	out := make([]string, 0, len(e.doc.Scenario.Artifacts.Required))
	for _, a := range e.doc.Scenario.Artifacts.Required {
		out = append(out, a.Type)
	}
	return out
}

// GatesBeforeStep returns gates scheduled before a step id executes.
func (e *Engine) GatesBeforeStep(stepID string) []Gate {
	if e.doc == nil {
		return nil
	}
	prefix := "before.step." + stepID
	var out []Gate
	for _, g := range e.doc.Scenario.Gates {
		if g.When == prefix {
			out = append(out, g)
		}
	}
	return out
}

// GatesBeforeFinish returns gates for run.before_finish.
func (e *Engine) GatesBeforeFinish() []Gate {
	if e.doc == nil {
		return nil
	}
	var out []Gate
	for _, g := range e.doc.Scenario.Gates {
		if g.When == "run.before_finish" {
			out = append(out, g)
		}
	}
	return out
}

// EvaluateHooks checks hook rules for an event; returns deny reason if blocked.
func (e *Engine) EvaluateHooks(eventType string, context map[string]any) (denied bool, reason string) {
	if e.doc == nil {
		return false, ""
	}
	for _, h := range e.doc.Hooks {
		if h.On != eventType || h.Policy != "enforce" {
			continue
		}
		for _, rule := range h.Rules {
			if matchAll(rule.Match, context) && rule.Action.Deny {
				return true, rule.Action.Reason
			}
		}
	}
	return false, ""
}

func matchAll(match, ctx map[string]any) bool {
	for k, v := range match {
		cv, ok := ctx[k]
		if !ok || fmt.Sprint(cv) != fmt.Sprint(v) {
			return false
		}
	}
	return true
}

// StepOrder returns step ids in definition order.
func (e *Engine) StepOrder() []string {
	if e.doc == nil {
		return nil
	}
	ids := make([]string, 0, len(e.doc.Scenario.Steps))
	for _, s := range e.doc.Scenario.Steps {
		ids = append(ids, s.ID)
	}
	return ids
}
