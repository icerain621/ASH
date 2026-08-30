package waker

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/ash-repwiki/ash/internal/runs"
	"github.com/ash-repwiki/ash/internal/store"
)

const (
	defaultRunTTL   = 2 * time.Hour
	defaultMaxItems = 50
)

// Item is one stale/stuck run candidate.
type Item struct {
	RunID     string `json:"runId"`
	SpaceID   string `json:"spaceId"`
	Status    string `json:"status"`
	AgeMs     int64  `json:"ageMs"`
	UpdatedAt int64  `json:"updatedAt"`
	Reason    string `json:"reason"`
}

// QueueResponse is GET /waker/queue.
type QueueResponse struct {
	Items     []Item    `json:"items"`
	Count     int       `json:"count"`
	MaxAge    string    `json:"maxAge"`
	MaxAgeMs  int64     `json:"maxAgeMs"`
	Inspected time.Time `json:"inspectedAt"`
}

// SweepRequest is POST /waker/sweep (report-only stub; dryRun default true).
type SweepRequest struct {
	SpaceID string `json:"spaceId,omitempty"`
	DryRun  *bool  `json:"dryRun,omitempty"`
	MaxAge  string `json:"maxAge,omitempty"`
	Limit   int    `json:"limit,omitempty"`
	ActorID string `json:"actorId,omitempty"`
}

// SweepResponse summarizes one wake pass.
type SweepResponse struct {
	OK      bool     `json:"ok"`
	DryRun  bool     `json:"dryRun"`
	Matched int      `json:"matched"`
	Flagged int      `json:"flagged"`
	RunIDs  []string `json:"runIds,omitempty"`
	MaxAge  string   `json:"maxAge"`
	Summary string   `json:"summary,omitempty"`
}

// Service inspects long-lived non-terminal runs (Sprint DX6).
type Service struct {
	db  *store.DB
	ctx context.Context
}

func NewService(db *store.DB) *Service {
	return &Service{db: db}
}

func (s *Service) WithContext(ctx context.Context) *Service {
	if s == nil || ctx == nil {
		return s
	}
	out := *s
	out.ctx = ctx
	if s.db != nil {
		out.db = s.db.BindContext(ctx)
	}
	return &out
}

func (s *Service) q() *gorm.DB {
	if s == nil || s.db == nil {
		return nil
	}
	if s.ctx != nil {
		return s.db.WithContext(s.ctx)
	}
	return s.db.DB
}

// EffectiveRunTTL parses ASH_WAKER_RUN_TTL (default 2h).
func EffectiveRunTTL() time.Duration {
	raw := strings.TrimSpace(os.Getenv("ASH_WAKER_RUN_TTL"))
	if raw == "" {
		return defaultRunTTL
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d < time.Minute {
		return defaultRunTTL
	}
	return d
}

// ParseInterval reads ASH_WAKER_INTERVAL; empty/off disables background.
func ParseInterval(raw string) (time.Duration, bool) {
	raw = strings.TrimSpace(strings.ToLower(raw))
	switch raw {
	case "", "0", "off", "false", "disable", "disabled":
		return 0, false
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d < time.Minute {
		return 0, false
	}
	return d, true
}

func parseMaxAge(raw string) time.Duration {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return EffectiveRunTTL()
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d < time.Minute {
		return EffectiveRunTTL()
	}
	return d
}

// Queue lists stale running / waiting_approval runs.
func (s *Service) Queue(spaceID, maxAge string, limit int) (QueueResponse, error) {
	ttl := parseMaxAge(maxAge)
	if limit <= 0 || limit > 200 {
		limit = defaultMaxItems
	}
	items, err := s.listStale(spaceID, ttl, limit)
	if err != nil {
		return QueueResponse{}, err
	}
	return QueueResponse{
		Items: items, Count: len(items),
		MaxAge: ttl.String(), MaxAgeMs: ttl.Milliseconds(),
		Inspected: time.Now().UTC(),
	}, nil
}

// Sweep reports stale runs. When dryRun=false, Flagged equals Matched (audit left to API).
func (s *Service) Sweep(req SweepRequest) (SweepResponse, error) {
	dryRun := true
	if req.DryRun != nil {
		dryRun = *req.DryRun
	}
	ttl := parseMaxAge(req.MaxAge)
	limit := req.Limit
	if limit <= 0 || limit > 200 {
		limit = defaultMaxItems
	}
	items, err := s.listStale(req.SpaceID, ttl, limit)
	if err != nil {
		return SweepResponse{}, err
	}
	out := SweepResponse{
		OK: true, DryRun: dryRun, Matched: len(items), MaxAge: ttl.String(),
	}
	for _, it := range items {
		out.RunIDs = append(out.RunIDs, it.RunID)
	}
	if !dryRun {
		out.Flagged = len(items)
		out.Summary = fmt.Sprintf("flagged %d stale runs for operator review", len(items))
	} else {
		out.Summary = fmt.Sprintf("matched %d stale runs (dryRun)", len(items))
	}
	return out, nil
}

func (s *Service) listStale(spaceID string, ttl time.Duration, limit int) ([]Item, error) {
	if s.q() == nil {
		return nil, fmt.Errorf("database unavailable")
	}
	cutoff := time.Now().UTC().Add(-ttl)
	q := s.q().Model(&store.RunRecord{}).
		Where("status IN ? AND updated_at < ?", []string{runs.StatusRunning, runs.StatusWaitingApproval}, cutoff)
	spaceID = strings.TrimSpace(spaceID)
	if spaceID != "" {
		q = q.Where("space_id = ?", spaceID)
	}
	var rows []store.RunRecord
	if err := q.Order("updated_at ASC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	items := make([]Item, 0, len(rows))
	for _, row := range rows {
		age := now.Sub(row.UpdatedAt)
		items = append(items, Item{
			RunID: row.ID, SpaceID: row.SpaceID, Status: row.Status,
			AgeMs: age.Milliseconds(), UpdatedAt: row.UpdatedAt.Unix(),
			Reason: "stale_" + row.Status,
		})
	}
	return items, nil
}

// StartBackground periodically sweeps all spaces (report-only, dryRun=false flags via logs).
func StartBackground(db *store.DB, interval time.Duration) context.CancelFunc {
	ctx, cancel := context.WithCancel(context.Background())
	if db == nil || interval <= 0 {
		return cancel
	}
	svc := NewService(db)
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		run := func() {
			dry := false
			resp, err := svc.Sweep(SweepRequest{DryRun: &dry, Limit: defaultMaxItems})
			if err != nil {
				log.Printf("waker: sweep: %v", err)
				return
			}
			if resp.Matched > 0 {
				log.Printf("waker: %s", resp.Summary)
			}
		}
		run()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				run()
			}
		}
	}()
	return cancel
}
