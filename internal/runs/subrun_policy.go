package runs

import (
	"github.com/ash-repwiki/ash/internal/spacepolicy"
	"github.com/ash-repwiki/ash/internal/store"
)

func (s *Service) spaceSubRunPolicy(spaceID string) spacepolicy.SubRunPolicy {
	spaceID = firstNonEmpty(spaceID, "local")
	var row store.SpacePolicyPack
	if err := s.gdb().Where("space_id = ?", spaceID).First(&row).Error; err != nil {
		return spacepolicy.SubRunPolicy{}
	}
	p, err := spacepolicy.SubRunFromBodyJSON(row.BodyJSON)
	if err != nil {
		return spacepolicy.SubRunPolicy{}
	}
	return p
}

func applySubRunDepth(harnessDepth int, policy spacepolicy.SubRunPolicy) int {
	if policy.MaxDepth > 0 {
		return policy.MaxDepth
	}
	if harnessDepth <= 0 {
		return 2
	}
	return harnessDepth
}

func applySubRunTools(requested []string, policy spacepolicy.SubRunPolicy) []string {
	base := normalizeAllowlist(requested)
	if len(base) == 0 {
		base = append([]string(nil), DefaultSubRunAllowlist...)
	}
	allowed := normalizeAllowlist(policy.AllowedTools)
	if len(allowed) == 0 {
		return base
	}
	keep := map[string]struct{}{}
	for _, t := range allowed {
		keep[t] = struct{}{}
	}
	var out []string
	for _, t := range base {
		if _, ok := keep[t]; ok {
			out = append(out, t)
		}
	}
	return out
}
