package spacepolicy

// QuotaUsage is live occupancy for DX68 projection.
type QuotaUsage struct {
	ActiveConcurrentRuns int `json:"activeConcurrentRuns"`
	TokenBudgetProxyUsed int `json:"tokenBudgetProxyUsed"`
}

// QuotaStatus is GET /spaces/{id}/quotas payload (limits + usage).
type QuotaStatus struct {
	SpaceID string     `json:"spaceId"`
	Limits  Quotas     `json:"limits"`
	Usage   QuotaUsage `json:"usage"`
}

// BuildQuotaStatus joins BodyJSON quotas with an active-run count.
// tokenBudgetProxyUsed stays 0 until token accounting ships (still expose the limit).
func BuildQuotaStatus(spaceID string, bodyJSON string, activeConcurrent int) (QuotaStatus, error) {
	q, err := QuotasFromBodyJSON(bodyJSON)
	if err != nil {
		return QuotaStatus{}, err
	}
	if activeConcurrent < 0 {
		activeConcurrent = 0
	}
	return QuotaStatus{
		SpaceID: spaceID,
		Limits:  q,
		Usage: QuotaUsage{
			ActiveConcurrentRuns: activeConcurrent,
			TokenBudgetProxyUsed: 0,
		},
	}, nil
}
