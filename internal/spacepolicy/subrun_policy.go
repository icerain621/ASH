package spacepolicy

import (
	"encoding/json"
	"fmt"
	"strings"
)

// SubRunPolicy is bodyJson.subRun (DX71). Zero maxDepth means "do not override Harness".
type SubRunPolicy struct {
	MaxDepth         int      `json:"maxDepth"`
	AllowedTools     []string `json:"allowedTools"`
	TokenBudgetProxy int      `json:"tokenBudgetProxy"`
}

type bodySubRunEnvelope struct {
	SubRun *SubRunPolicy `json:"subRun"`
}

// SubRunFromBodyJSON reads subRun policy. Missing → zero policy (Harness unchanged).
func SubRunFromBodyJSON(bodyJSON string) (SubRunPolicy, error) {
	bodyJSON = strings.TrimSpace(bodyJSON)
	if bodyJSON == "" || bodyJSON == "{}" {
		return SubRunPolicy{}, nil
	}
	var env bodySubRunEnvelope
	if err := json.Unmarshal([]byte(bodyJSON), &env); err != nil {
		return SubRunPolicy{}, fmt.Errorf("bodyJson subRun: %w", err)
	}
	if env.SubRun == nil {
		return SubRunPolicy{}, nil
	}
	p := *env.SubRun
	if p.MaxDepth < 0 {
		return SubRunPolicy{}, fmt.Errorf("subRun.maxDepth must be >= 0")
	}
	if p.TokenBudgetProxy < 0 {
		return SubRunPolicy{}, fmt.Errorf("subRun.tokenBudgetProxy must be >= 0")
	}
	return p, nil
}
