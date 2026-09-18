package runs

import (
	"strings"

	"github.com/ash-repwiki/ash/internal/execpolicy"
	"github.com/ash-repwiki/ash/internal/sandbox"
	"github.com/ash-repwiki/ash/internal/store"
)

// loadSpaceExecPolicy reads ash.execpolicy.v1 from SpacePolicy BodyJSON.
// Missing pack / empty execPolicy → empty Policy (no floor; does not lower isolation).
func (s *Service) loadSpaceExecPolicy(spaceID string) execpolicy.Policy {
	if s == nil {
		return execpolicy.Policy{}
	}
	spaceID = firstNonEmpty(strings.TrimSpace(spaceID), "local")
	var row store.SpacePolicyPack
	if err := s.gdb().First(&row, "space_id = ?", spaceID).Error; err != nil {
		return execpolicy.Policy{}
	}
	p, err := execpolicy.FromSpaceBodyJSON(row.BodyJSON)
	if err != nil {
		// Fail soft for resolve path: invalid body should have been rejected on Put;
		// treat as empty so we never lower isolation unexpectedly.
		return execpolicy.Policy{}
	}
	return p
}

// spaceExecPolicyFloor returns the sandbox mode floor from SpacePolicy execPolicy.
func (s *Service) spaceExecPolicyFloor(spaceID string) string {
	return sandbox.FloorFromExecPolicy(s.loadSpaceExecPolicy(spaceID))
}
