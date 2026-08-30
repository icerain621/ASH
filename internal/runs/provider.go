package runs

import (
	"context"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ash-repwiki/ash/internal/agentexec"
)

// ProviderSelection describes how the agent adapter was chosen for a run.
type ProviderSelection struct {
	RequestedKind string
	Adapter       string
	Source        string // pinned | harness | service_default
	Fallback      bool
	Reason        string
	Executor      agentexec.Executor
}

type providerProbeCache struct {
	mu        sync.Mutex
	execGo    agentexec.ProbeReport
	execGoAt  time.Time
	acp       agentexec.ProbeReport
	acpAt     time.Time
}

func (s *Service) probeExecGoCached() agentexec.ProbeReport {
	const ttl = 30 * time.Second
	if s == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		return agentexec.ProbeExecGo(ctx)
	}
	if s.providerProbe == nil {
		s.providerProbe = &providerProbeCache{}
	}
	c := s.providerProbe
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.execGoAt.IsZero() && time.Since(c.execGoAt) < ttl {
		return c.execGo
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	c.execGo = agentexec.ProbeExecGo(ctx)
	c.execGoAt = time.Now()
	return c.execGo
}

func (s *Service) probeACPCached() agentexec.ProbeReport {
	const ttl = 30 * time.Second
	if s == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		return agentexec.ProbeACP(ctx)
	}
	if s.providerProbe == nil {
		s.providerProbe = &providerProbeCache{}
	}
	c := s.providerProbe
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.acpAt.IsZero() && time.Since(c.acpAt) < ttl {
		return c.acp
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	c.acp = agentexec.ProbeACP(ctx)
	c.acpAt = time.Now()
	return c.acp
}

// SelectProvider resolves the agent executor for a space.
// CLI / WithAgentExecutor pins win; otherwise active Harness provider.kind is used.
// execgo / acp_sdk unhealthy → static + Fallback=true.
func (s *Service) SelectProvider(spaceID string) ProviderSelection {
	if s == nil {
		return ProviderSelection{
			RequestedKind: "static", Adapter: "static", Source: "service_default",
			Executor: agentexec.StaticExecutor{},
		}
	}
	if s.agentPinned && s.agent != nil {
		adapter := s.AgentAdapter()
		return ProviderSelection{
			RequestedKind: adapter, Adapter: adapter, Source: "pinned",
			Executor: s.agent,
		}
	}

	kind := "execgo"
	source := "service_default"
	if s.harnessSvc != nil {
		if view, err := s.harnessSvc.LoadActive(firstNonEmpty(spaceID, "local"), "default"); err == nil && view != nil {
			if k := strings.TrimSpace(view.Spec.Provider.Kind); k != "" {
				kind = agentexec.NormalizeProviderKind(k)
				source = "harness"
			}
		}
	}

	exec := agentexec.Resolve(kind)
	sel := ProviderSelection{
		RequestedKind: kind,
		Adapter:       agentexec.AdapterNameOf(exec),
		Source:        source,
		Executor:      exec,
	}

	switch kind {
	case "acp_sdk":
		probe := s.probeACPCached()
		if !probe.OK {
			sel.Fallback = true
			sel.Reason = probe.Message
			if sel.Reason == "" {
				sel.Reason = "acp_sdk probe failed"
			}
			sel.Executor = agentexec.StaticExecutor{}
			sel.Adapter = "static"
		}
	case "execgo":
		probe := s.probeExecGoCached()
		if !probe.OK {
			sel.Fallback = true
			sel.Reason = probe.Message
			if sel.Reason == "" {
				sel.Reason = "execgo probe failed"
			}
			sel.Executor = agentexec.StaticExecutor{}
			sel.Adapter = "static"
		}
	}
	return sel
}

// AgentProviderStatus is the GET /providers/agent response body.
type AgentProviderStatus struct {
	Pinned           bool                  `json:"pinned"`
	PinnedAdapter    string                `json:"pinnedAdapter,omitempty"`
	HarnessKind      string                `json:"harnessKind,omitempty"`
	Selection        ProviderSelectionDTO  `json:"selection"`
	ExecGo           agentexec.ProbeReport `json:"execGo"`
	ACP              agentexec.ProbeReport `json:"acp"`
	LiveGateHints    []string              `json:"liveGateHints,omitempty"`
	ExecGoE2EEnabled bool                  `json:"execGoE2EEnabled"`
	ACPE2EEnabled    bool                  `json:"acpE2EEnabled"`
	LiveSmokeHint    string                `json:"liveSmokeHint,omitempty"`
}

// ProviderSelectionDTO is JSON-safe selection without the executor.
type ProviderSelectionDTO struct {
	RequestedKind string `json:"requestedKind"`
	Adapter       string `json:"adapter"`
	Source        string `json:"source"`
	Fallback      bool   `json:"fallback"`
	Reason        string `json:"reason,omitempty"`
}

// ProviderStatus builds a readiness snapshot for the probe API.
func (s *Service) ProviderStatus(spaceID string, liveHints []string, execGoE2E bool) AgentProviderStatus {
	harnessKind := ""
	if s != nil && s.harnessSvc != nil {
		if view, err := s.harnessSvc.LoadActive(firstNonEmpty(spaceID, "local"), "default"); err == nil && view != nil {
			harnessKind = agentexec.NormalizeProviderKind(view.Spec.Provider.Kind)
		}
	}
	sel := s.SelectProvider(spaceID)
	probe := s.probeExecGoCached()
	acpProbe := s.probeACPCached()
	out := AgentProviderStatus{
		Pinned:      s != nil && s.agentPinned,
		HarnessKind: harnessKind,
		Selection: ProviderSelectionDTO{
			RequestedKind: sel.RequestedKind,
			Adapter:       sel.Adapter,
			Source:        sel.Source,
			Fallback:      sel.Fallback,
			Reason:        sel.Reason,
		},
		ExecGo:           probe,
		ACP:              acpProbe,
		LiveGateHints:    liveHints,
		ExecGoE2EEnabled: execGoE2E,
		ACPE2EEnabled:    os.Getenv("ASH_ACP_E2E") == "1",
		LiveSmokeHint:    "ASH_EXECGO_E2E=1 make execgo-live-smoke; ACP: ASH_ACP_ENDPOINT + harness provider.kind=acp_sdk",
	}
	if s != nil && s.agentPinned {
		out.PinnedAdapter = s.AgentAdapter()
	}
	return out
}
