package spacepolicy

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Quotas is bodyJson.quotas (DX67). Zero / omit means unlimited.
type Quotas struct {
	MaxConcurrentRuns int `json:"maxConcurrentRuns"`
	TokenBudgetProxy  int `json:"tokenBudgetProxy"`
}

type bodyQuotasEnvelope struct {
	Quotas *Quotas `json:"quotas"`
}

// QuotasFromBodyJSON reads quotas from SpacePolicy BodyJSON. Missing → unlimited zeros.
func QuotasFromBodyJSON(bodyJSON string) (Quotas, error) {
	bodyJSON = strings.TrimSpace(bodyJSON)
	if bodyJSON == "" || bodyJSON == "{}" {
		return Quotas{}, nil
	}
	var env bodyQuotasEnvelope
	if err := json.Unmarshal([]byte(bodyJSON), &env); err != nil {
		return Quotas{}, fmt.Errorf("bodyJson quotas: %w", err)
	}
	if env.Quotas == nil {
		return Quotas{}, nil
	}
	q := *env.Quotas
	if q.MaxConcurrentRuns < 0 {
		return Quotas{}, fmt.Errorf("quotas.maxConcurrentRuns must be >= 0")
	}
	if q.TokenBudgetProxy < 0 {
		return Quotas{}, fmt.Errorf("quotas.tokenBudgetProxy must be >= 0")
	}
	return q, nil
}

// StricterQuotas picks the tighter positive limits (0 = unlimited on that side).
func StricterQuotas(a, b Quotas) Quotas {
	return Quotas{
		MaxConcurrentRuns: stricterPositive(a.MaxConcurrentRuns, b.MaxConcurrentRuns),
		TokenBudgetProxy:  stricterPositive(a.TokenBudgetProxy, b.TokenBudgetProxy),
	}
}

func stricterPositive(a, b int) int {
	switch {
	case a <= 0 && b <= 0:
		return 0
	case a <= 0:
		return b
	case b <= 0:
		return a
	case a < b:
		return a
	default:
		return b
	}
}
