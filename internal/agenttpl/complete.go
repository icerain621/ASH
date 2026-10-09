package agenttpl

import "github.com/ash-repwiki/ash/internal/modelrouter"

// Complete calls the provider port. Templates do not import a vendor SDK.
func Complete(port modelrouter.Completer, prompt string) modelrouter.Decision {
	if port == nil {
		return modelrouter.Decision{Status: "unavailable", Reason: "provider port is nil"}
	}
	return port.Complete(modelrouter.Request{Prompt: prompt, UseCase: "agent"})
}
