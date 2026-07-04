package memory

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ash-repwiki/ash/internal/store"
)

const BuiltinTTLReviewLeadDays = 7

// TTLRecordState classifies approved records with ttl_days.
type TTLRecordState string

const (
	TTLStateActive    TTLRecordState = "active"
	TTLStateReviewDue TTLRecordState = "review_due"
	TTLStateExpired   TTLRecordState = "expired"
)

// EffectiveTTLReviewLeadDays returns days-before-expiry review window (ASH_MEMORY_TTL_REVIEW_DAYS).
func EffectiveTTLReviewLeadDays() int {
	return effectiveTTLDays("ASH_MEMORY_TTL_REVIEW_DAYS", BuiltinTTLReviewLeadDays)
}

// TTLQueueItem is a memory record due for human review before TTL expiry.
type TTLQueueItem struct {
	RecordID      string `json:"recordId"`
	Layer         string `json:"layer"`
	Title         string `json:"title"`
	Status        string `json:"status"`
	DaysRemaining int    `json:"daysRemaining"`
	ExpiresAtMs   int64  `json:"expiresAtMs"`
}

// TTLQueueResponse summarizes TTL governance backlog for a space.
type TTLQueueResponse struct {
	ReviewDue             []TTLQueueItem `json:"reviewDue"`
	ReviewDueCount        int64          `json:"reviewDueCount"`
	ExpiredPendingCount   int64          `json:"expiredPendingCount"`
	ReviewLeadDays        int            `json:"reviewLeadDays"`
}

// SweepTTLRequest deprecates expired approved records and reports review-due backlog.
type SweepTTLRequest struct {
	SpaceID string `json:"spaceId,omitempty"`
	RunID   string `json:"runId,omitempty"`
	TraceID string `json:"traceId,omitempty"`
	ActorID string `json:"actorId,omitempty"`
	DryRun  bool   `json:"dryRun,omitempty"`
}

// SweepTTLResponse summarizes one TTL sweep batch.
type SweepTTLResponse struct {
	OK          bool   `json:"ok"`
	Deprecated  int    `json:"deprecated"`
	ReviewDue   int    `json:"reviewDue"`
	DryRun      bool   `json:"dryRun,omitempty"`
	Summary     string `json:"summary,omitempty"`
}

func classifyTTL(row store.MemoryRecord, now time.Time) TTLRecordState {
	if row.Status != "approved" {
		return TTLStateActive
	}
	expiresAt, ok := recordExpiresAt(row)
	if !ok {
		return TTLStateActive
	}
	if !now.Before(expiresAt) {
		return TTLStateExpired
	}
	lead := time.Duration(EffectiveTTLReviewLeadDays()) * 24 * time.Hour
	if expiresAt.Sub(now) <= lead {
		return TTLStateReviewDue
	}
	return TTLStateActive
}

func recordExpiresAt(row store.MemoryRecord) (time.Time, bool) {
	if row.TTLDays == nil || *row.TTLDays <= 0 {
		return time.Time{}, false
	}
	return row.CreatedAt.Add(time.Duration(*row.TTLDays) * 24 * time.Hour), true
}

func daysRemaining(expiresAt, now time.Time) int {
	if !now.Before(expiresAt) {
		return 0
	}
	hours := expiresAt.Sub(now).Hours()
	days := int(hours / 24)
	if hours > float64(days*24) {
		days++
	}
	if days < 0 {
		return 0
	}
	return days
}

// TTLCounts returns review-due and expired-pending approved records for a space.
func TTLCounts(db *store.DB, spaceID string) (reviewDue, expiredPending int64, err error) {
	if db == nil {
		return 0, 0, fmt.Errorf("database is nil")
	}
	spaceID = firstNonEmpty(spaceID, "local")
	now := time.Now().UTC()
	var rows []store.MemoryRecord
	q := db.Where("space_id = ? AND status = ? AND ttl_days IS NOT NULL AND ttl_days > 0", spaceID, "approved")
	if err := q.Find(&rows).Error; err != nil {
		return 0, 0, err
	}
	for _, row := range rows {
		switch classifyTTL(row, now) {
		case TTLStateReviewDue:
			reviewDue++
		case TTLStateExpired:
			expiredPending++
		}
	}
	return reviewDue, expiredPending, nil
}

// TTLQueue loads review-due items (up to limit) and counts expired-pending rows.
func TTLQueue(db *store.DB, spaceID string, limit int) (*TTLQueueResponse, error) {
	if db == nil {
		return nil, fmt.Errorf("database is nil")
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	spaceID = firstNonEmpty(spaceID, "local")
	now := time.Now().UTC()
	var rows []store.MemoryRecord
	if err := db.Where("space_id = ? AND status = ? AND ttl_days IS NOT NULL AND ttl_days > 0", spaceID, "approved").
		Order("created_at asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	resp := &TTLQueueResponse{
		ReviewDue:      make([]TTLQueueItem, 0, limit),
		ReviewLeadDays: EffectiveTTLReviewLeadDays(),
	}
	for _, row := range rows {
		state := classifyTTL(row, now)
		switch state {
		case TTLStateReviewDue:
			resp.ReviewDueCount++
			if len(resp.ReviewDue) < limit {
				expiresAt, _ := recordExpiresAt(row)
				resp.ReviewDue = append(resp.ReviewDue, TTLQueueItem{
					RecordID:      row.ID,
					Layer:         row.Layer,
					Title:         row.Title,
					Status:        string(TTLStateReviewDue),
					DaysRemaining: daysRemaining(expiresAt, now),
					ExpiresAtMs:   expiresAt.UTC().UnixMilli(),
				})
			}
		case TTLStateExpired:
			resp.ExpiredPendingCount++
		}
	}
	return resp, nil
}

// SweepTTL marks expired approved records deprecated and emits ttl_expired/deprecated signals.
func (s *Service) SweepTTL(req SweepTTLRequest) (*SweepTTLResponse, error) {
	spaceID := firstNonEmpty(req.SpaceID, "local")
	now := time.Now().UTC()
	traceID := req.TraceID
	if req.RunID != "" {
		var err error
		traceID, err = s.validateRunRef(req.RunID, req.TraceID)
		if err != nil {
			return nil, err
		}
	}

	var rows []store.MemoryRecord
	if err := s.gdb().
		Where("space_id = ? AND status = ? AND ttl_days IS NOT NULL AND ttl_days > 0", spaceID, "approved").
		Find(&rows).Error; err != nil {
		return nil, err
	}

	deprecated := 0
	reviewDue := 0
	for _, row := range rows {
		switch classifyTTL(row, now) {
		case TTLStateReviewDue:
			reviewDue++
		case TTLStateExpired:
			if req.DryRun {
				deprecated++
				continue
			}
			if err := s.deprecateExpiredRecord(row, spaceID, req.RunID, traceID, req.ActorID, now); err != nil {
				return nil, err
			}
			deprecated++
		}
	}

	summary := fmt.Sprintf("ttl sweep: deprecated=%d reviewDue=%d leadDays=%d", deprecated, reviewDue, EffectiveTTLReviewLeadDays())
	resp := &SweepTTLResponse{
		OK:         true,
		Deprecated: deprecated,
		ReviewDue:  reviewDue,
		DryRun:     req.DryRun,
		Summary:    summary,
	}
	if req.DryRun {
		resp.Summary = "dry-run: " + summary
	}
	return resp, nil
}

func (s *Service) deprecateExpiredRecord(row store.MemoryRecord, spaceID, runID, traceID, actorID string, now time.Time) error {
	reason := "ttl_expired"
	return s.gdb().Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&store.MemoryRecord{}).
			Where("id = ? AND status = ?", row.ID, "approved").
			Updates(map[string]any{"status": "deprecated", "updated_at": now})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return nil
		}
		payload, _ := json.Marshal(map[string]any{
			"memoryId": row.ID, "layer": row.Layer, "reason": reason, "ttlDays": row.TTLDays,
		})
		if err := tx.Create(&store.AuditLog{
			ID: "aud_" + uuid.NewString(), SpaceID: spaceID, EventType: "memory.ttl_expired",
			ActorID: firstNonEmpty(actorID, "ash-memory"), PayloadJSON: string(payload), CreatedAt: now,
		}).Error; err != nil {
			return err
		}
		depPayload, _ := json.Marshal(map[string]any{
			"memoryId": row.ID, "layer": row.Layer, "reason": reason,
		})
		if err := tx.Create(&store.AuditLog{
			ID: "aud_" + uuid.NewString(), SpaceID: spaceID, EventType: "memory.deprecated",
			ActorID: firstNonEmpty(actorID, "ash-memory"), PayloadJSON: string(depPayload), CreatedAt: now,
		}).Error; err != nil {
			return err
		}
		return s.emitTTLExpired(runID, traceID, row.ID, row.Layer, reason)
	})
}
