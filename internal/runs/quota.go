package runs

import (
	"errors"
	"fmt"

	"github.com/ash-repwiki/ash/internal/spacepolicy"
	"github.com/ash-repwiki/ash/internal/store"
)

// ErrSpaceQuotaExceeded is returned before a run row is created when space quotas block Create/Spawn.
var ErrSpaceQuotaExceeded = errors.New("SPACE_QUOTA_EXCEEDED")

// enforceSpaceQuotas fails closed when maxConcurrentRuns is set and active runs are at limit.
// tokenBudgetProxy is parsed for DX68 projection; hard token accounting is not enforced here.
func (s *Service) enforceSpaceQuotas(spaceID string) error {
	spaceID = firstNonEmpty(spaceID, "local")
	var row store.SpacePolicyPack
	if err := s.gdb().Where("space_id = ?", spaceID).First(&row).Error; err != nil {
		// No pack → unlimited (same as missing quotas).
		return nil
	}
	q, err := spacepolicy.QuotasFromBodyJSON(row.BodyJSON)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrSpaceQuotaExceeded, err)
	}
	if q.MaxConcurrentRuns <= 0 {
		return nil
	}
	n, err := s.CountActiveRuns(spaceID)
	if err != nil {
		return err
	}
	if int(n) >= q.MaxConcurrentRuns {
		return fmt.Errorf("%w: concurrent runs %d >= maxConcurrentRuns %d", ErrSpaceQuotaExceeded, n, q.MaxConcurrentRuns)
	}
	return nil
}

// CountActiveRuns returns runs in space with status running or waiting_approval (DX68 usage).
func (s *Service) CountActiveRuns(spaceID string) (int64, error) {
	spaceID = firstNonEmpty(spaceID, "local")
	var n int64
	if err := s.gdb().Model(&store.RunRecord{}).
		Where("space_id = ? AND status IN ?", spaceID, []string{StatusRunning, StatusWaitingApproval}).
		Count(&n).Error; err != nil {
		return 0, err
	}
	return n, nil
}
