package evolve

import "strings"

// Allowed feedback target types (appendix K §3).
var AllowedTargetTypes = map[string]struct{}{
	"memory": {}, "memory_hit": {},
	"run": {}, "run_step": {},
	"plan": {}, "artifact": {},
	"skill": {}, "harness_profile": {}, "scenario_patch": {},
	// v1 compatibility
	"ci_diagnosis": {}, "release": {},
}

func NormalizeTargetType(v string) (string, bool) {
	v = strings.ToLower(strings.TrimSpace(v))
	_, ok := AllowedTargetTypes[v]
	return v, ok
}

const (
	QueueMemory        = "memory"
	QueueOrchestration = "orchestration"
	StatusPending      = "pending"
	StatusApproved     = "approved"
	StatusRejected     = "rejected"

	DecisionApprove = "approve"
	DecisionReject  = "reject"
)

// ItemID encodes queue item identity: "memory:<id>" | "harness_profile:<id>"
func ItemID(targetType, targetID string) string {
	return strings.TrimSpace(targetType) + ":" + strings.TrimSpace(targetID)
}

func ParseItemID(id string) (targetType, targetID string, ok bool) {
	id = strings.TrimSpace(id)
	i := strings.IndexByte(id, ':')
	if i <= 0 || i == len(id)-1 {
		return "", "", false
	}
	return id[:i], id[i+1:], true
}
