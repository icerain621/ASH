package session

import (
	"context"
	"strings"
	"time"

	"github.com/ash-repwiki/ash/internal/agentexec"
)

// forwardTurnACP best-effort posts the turn prompt to ACP when the session is ACP-bound and healthy.
func (s *Service) forwardTurnACP(view *View, turn Turn) map[string]any {
	if view == nil {
		return nil
	}
	kind := agentexec.NormalizeProviderKind(view.ProviderKind)
	if kind != "acp_sdk" {
		return nil
	}
	if view.ProviderFallback && view.ProviderAdapter != "acp_sdk" {
		return map[string]any{"acpForwarded": false, "acpSkipped": "provider fallback"}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	probe := agentexec.ProbeACP(ctx)
	if !probe.OK {
		return map[string]any{"acpForwarded": false, "acpSkipped": probe.Message}
	}
	exec := agentexec.NewACPExecutor()
	res, err := exec.Execute(ctx, agentexec.Request{
		RunID: view.RunID, TraceID: view.TraceID, StepID: turn.ID,
		RepoRoot: view.RepoRoot, Prompt: turn.Prompt,
		Metadata: map[string]any{"sessionId": view.ID},
		TimeoutMs: 30000,
	})
	if err != nil {
		return map[string]any{"acpForwarded": false, "acpError": err.Error()}
	}
	if view.Meta == nil {
		view.Meta = map[string]any{}
	}
	view.Meta["lastAcpTaskId"] = res.TaskID
	view.Meta["lastAcpStatus"] = res.Status
	out := map[string]any{
		"acpForwarded": true,
		"acpTaskId":    res.TaskID,
		"acpStatus":    res.Status,
		"acpAdapter":   res.Adapter,
	}
	if strings.TrimSpace(res.StdoutSummary) != "" {
		out["acpMessage"] = res.StdoutSummary
	}
	return out
}
