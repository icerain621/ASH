package session

import (
	"context"
	"strings"
	"time"

	"github.com/ash-repwiki/ash/internal/agentexec"
	"github.com/ash-repwiki/ash/internal/store"
)

// ProviderBinding mirrors run-side provider selection for session documents.
type ProviderBinding struct {
	Kind     string
	Adapter  string
	Fallback bool
	Reason   string
}

// FindByRunID returns the newest active agent.session bound to runID (if any).
func (s *Service) FindByRunID(spaceID, runID string) (*View, error) {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return nil, nil
	}
	space := firstNonEmpty(strings.TrimSpace(spaceID), "local")
	var row store.AuditLog
	err := s.q().Where("event_type = ? AND run_id = ? AND space_id = ?", auditEventType, runID, space).
		Order("created_at DESC").First(&row).Error
	if err != nil {
		return nil, nil
	}
	view, err := decodeView(row)
	if err != nil {
		return nil, err
	}
	view.StreamURL = streamURL(view.RunID)
	return view, nil
}

// EnsureForRun finds or creates a session for the run and applies provider binding.
// created=true when a new session document was inserted.
func (s *Service) EnsureForRun(spaceID, runID, repoRoot, createdBy string, bind ProviderBinding) (*View, bool, error) {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return nil, false, nil
	}
	space := firstNonEmpty(strings.TrimSpace(spaceID), "local")
	existing, err := s.FindByRunID(space, runID)
	if err != nil {
		return nil, false, err
	}
	if existing != nil {
		applyProviderBinding(existing, bind)
		existing.UpdatedAt = time.Now().UTC().Unix()
		if existing.RepoRoot == "" {
			existing.RepoRoot = strings.TrimSpace(repoRoot)
		}
		if err := s.save(existing); err != nil {
			return nil, false, err
		}
		return existing, false, nil
	}
	view, err := s.Create(CreateRequest{
		RunID: runID, RepoRoot: repoRoot, SpaceID: space,
		CreatedBy: firstNonEmpty(createdBy, "runs"), ProviderKind: bind.Kind,
	})
	if err != nil {
		return nil, false, err
	}
	// Create already probed from kind; overwrite with authoritative run selection when provided.
	if bind.Kind != "" || bind.Adapter != "" {
		applyProviderBinding(view, bind)
		if err := s.save(view); err != nil {
			return nil, false, err
		}
	}
	return view, true, nil
}

func (s *Service) applyProviderKind(view *View, kind string) {
	if view == nil {
		return
	}
	kind = agentexec.NormalizeProviderKind(strings.TrimSpace(kind))
	if kind == "" {
		return
	}
	bind := ProviderBinding{Kind: kind}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	switch kind {
	case "acp_sdk":
		probe := agentexec.ProbeACP(ctx)
		if probe.OK {
			bind.Adapter = "acp_sdk"
		} else {
			bind.Adapter = "static"
			bind.Fallback = true
			bind.Reason = probe.Message
		}
	case "execgo":
		probe := agentexec.ProbeExecGo(ctx)
		if probe.OK {
			bind.Adapter = "execgo_codex"
		} else {
			bind.Adapter = "static"
			bind.Fallback = true
			bind.Reason = probe.Message
		}
	case "static":
		bind.Adapter = "static"
	default:
		bind.Adapter = "static"
		bind.Fallback = true
		bind.Reason = "unknown provider kind"
	}
	applyProviderBinding(view, bind)
}

func applyProviderBinding(view *View, bind ProviderBinding) {
	if view == nil {
		return
	}
	if k := agentexec.NormalizeProviderKind(bind.Kind); k != "" {
		view.ProviderKind = k
	}
	if bind.Adapter != "" {
		view.ProviderAdapter = bind.Adapter
	}
	view.ProviderFallback = bind.Fallback
	view.ProviderReason = bind.Reason
	if view.Meta == nil {
		view.Meta = map[string]any{}
	}
	if view.ProviderKind != "" {
		view.Meta["providerKind"] = view.ProviderKind
	}
	if view.ProviderAdapter != "" {
		view.Meta["providerAdapter"] = view.ProviderAdapter
	}
	view.Meta["providerFallback"] = view.ProviderFallback
	if view.ProviderReason != "" {
		view.Meta["providerReason"] = view.ProviderReason
	}
}
