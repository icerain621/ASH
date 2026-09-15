package session

import (
	"context"
	"strings"
	"time"

	"github.com/ash-repwiki/ash/internal/agentexec"
)

// forwardTurnProvider best-effort runs the session provider executor and returns ledger metadata.
// Empty ProviderKind skips (caller falls back to echo). Probe failures skip without error.
func (s *Service) forwardTurnProvider(view *View, turn Turn) map[string]any {
	if view == nil {
		return nil
	}
	kind := agentexec.NormalizeProviderKind(view.ProviderKind)
	if kind == "" {
		return nil
	}
	// Already fell back at bind time — do not re-hit a known-unhealthy bridge.
	if view.ProviderFallback && view.ProviderAdapter == "static" && kind != "static" {
		return map[string]any{
			"providerForwarded": false,
			"providerSkipped":   "provider fallback",
			"providerKind":      kind,
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	switch kind {
	case "acp_sdk":
		probe := agentexec.ProbeACP(ctx)
		if !probe.OK {
			return map[string]any{
				"providerForwarded": false,
				"providerSkipped":   probe.Message,
				"providerKind":      kind,
			}
		}
	case "execgo":
		probe := agentexec.ProbeExecGo(ctx)
		if !probe.OK {
			return map[string]any{
				"providerForwarded": false,
				"providerSkipped":   probe.Message,
				"providerKind":      kind,
			}
		}
	case "static":
		// always ok
	}

	resolveKind := kind
	if view.ProviderAdapter != "" && !view.ProviderFallback {
		// Prefer bound adapter when it matches a resolve key (static / acp_sdk / execgo*).
		switch view.ProviderAdapter {
		case "static", "acp_sdk":
			resolveKind = view.ProviderAdapter
		case "execgo_codex":
			resolveKind = "execgo"
		}
	}

	exec := agentexec.Resolve(resolveKind)
	adapter := agentexec.AdapterNameOf(exec)
	res, err := exec.Execute(ctx, agentexec.Request{
		RunID: view.RunID, TraceID: view.TraceID, StepID: turn.ID,
		RepoRoot: view.RepoRoot, Prompt: turn.Prompt,
		Metadata:  map[string]any{"sessionId": view.ID},
		TimeoutMs: 30000,
	})
	if err != nil {
		return map[string]any{
			"providerForwarded": false,
			"providerError":     err.Error(),
			"providerKind":      kind,
			"providerAdapter":   adapter,
		}
	}
	if view.Meta == nil {
		view.Meta = map[string]any{}
	}
	view.Meta["lastProviderTaskId"] = res.TaskID
	view.Meta["lastProviderStatus"] = res.Status
	view.Meta["lastProviderAdapter"] = firstNonEmpty(res.Adapter, adapter)
	// Keep legacy ACP meta keys when ACP ran successfully (compat with older tests/UI).
	if kind == "acp_sdk" && !view.ProviderFallback {
		view.Meta["lastAcpTaskId"] = res.TaskID
		view.Meta["lastAcpStatus"] = res.Status
	}
	out := map[string]any{
		"providerForwarded": true,
		"providerTaskId":    res.TaskID,
		"providerStatus":    res.Status,
		"providerAdapter":   firstNonEmpty(res.Adapter, adapter),
		"providerKind":      kind,
	}
	if strings.TrimSpace(res.StdoutSummary) != "" {
		out["providerMessage"] = res.StdoutSummary
		if kind == "acp_sdk" {
			out["acpMessage"] = res.StdoutSummary
			out["acpForwarded"] = true
			out["acpTaskId"] = res.TaskID
			out["acpStatus"] = res.Status
			out["acpAdapter"] = firstNonEmpty(res.Adapter, adapter)
		}
	}
	return out
}
