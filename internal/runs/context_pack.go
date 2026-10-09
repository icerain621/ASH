package runs

import (
	"fmt"
	"strings"

	"github.com/ash-repwiki/ash/internal/contextpack"
)

func evidenceWithoutMemory(refs []string) []string {
	out := make([]string, 0, len(refs))
	for _, ref := range refs {
		if strings.HasPrefix(ref, "memory:") {
			continue
		}
		out = append(out, ref)
	}
	return out
}

func memoryHitsFromRefs(refs []string) []contextpack.Hit {
	var hits []contextpack.Hit
	seen := map[string]struct{}{}
	for _, ref := range refs {
		id := strings.TrimPrefix(ref, "memory:")
		if id == ref || id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		hits = append(hits, contextpack.Hit{ID: id})
	}
	return hits
}

func packWithinEvidence(packRefs, evidence []string) error {
	have := map[string]struct{}{}
	for _, ref := range evidence {
		have[ref] = struct{}{}
	}
	for _, ref := range packRefs {
		if _, ok := have[ref]; !ok {
			return fmt.Errorf("context pack ref %s is not in evidence", ref)
		}
	}
	return nil
}
