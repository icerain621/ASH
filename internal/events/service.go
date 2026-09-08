package events

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/ash-repwiki/ash/internal/store"
	"gorm.io/gorm"
)

type Envelope struct {
	ID       string          `json:"id"`
	TraceID  string          `json:"traceId"`
	RunID    string          `json:"runId"`
	Seq      int64           `json:"seq"`
	TS       int64           `json:"ts"`
	Type     string          `json:"type"`
	Severity string          `json:"severity"`
	Payload  json.RawMessage `json:"payload"`
}

type Service struct {
	db  *store.DB
	ctx context.Context
}

func NewService(db *store.DB) *Service {
	return &Service{db: db}
}

// WithContext returns a shallow copy bound to ctx for Postgres RLS session vars.
func (s *Service) WithContext(ctx context.Context) *Service {
	if s == nil || ctx == nil {
		return s
	}
	return &Service{db: s.db, ctx: ctx}
}

func (s *Service) gdb() *gorm.DB {
	if s == nil || s.db == nil {
		return nil
	}
	if s.ctx != nil {
		return s.db.WithContext(s.ctx)
	}
	return s.db.DB
}

func (s *Service) Append(runID, traceID, eventType, severity string, payload any) (*Envelope, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}
	if PayloadValidationEnabled() {
		if err := ValidatePayload(eventType, payloadBytes); err != nil {
			return nil, err
		}
	}

	var env Envelope
	err = s.gdb().Transaction(func(tx *gorm.DB) error {
		var last store.RunEvent
		q := tx.Where("run_id = ?", runID).Order("seq desc").Limit(1).First(&last)
		seq := int64(1)
		if q.Error == nil {
			seq = last.Seq + 1
		} else if q.Error != gorm.ErrRecordNotFound {
			return q.Error
		}

		now := time.Now().UTC()
		rec := store.RunEvent{
			ID:          "evt_" + uuid.NewString(),
			RunID:       runID,
			Seq:         seq,
			TS:          now.UnixMilli(),
			Type:        eventType,
			Severity:    severity,
			PayloadJSON: string(payloadBytes),
			CreatedAt:   now,
		}
		if err := tx.Create(&rec).Error; err != nil {
			return err
		}
		env = Envelope{
			ID:       rec.ID,
			TraceID:  traceID,
			RunID:    runID,
			Seq:      rec.Seq,
			TS:       rec.TS,
			Type:     rec.Type,
			Severity: rec.Severity,
			Payload:  json.RawMessage(rec.PayloadJSON),
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &env, nil
}

func (s *Service) ListAfter(runID string, afterSeq int64, limit int) ([]Envelope, error) {
	if limit <= 0 {
		limit = 500
	}
	var rows []store.RunEvent
	q := s.gdb().Where("run_id = ? AND seq > ?", runID, afterSeq).Order("seq asc").Limit(limit).Find(&rows)
	if q.Error != nil {
		return nil, q.Error
	}
	out := make([]Envelope, 0, len(rows))
	for _, r := range rows {
		out = append(out, Envelope{
			ID:       r.ID,
			RunID:    r.RunID,
			Seq:      r.Seq,
			TS:       r.TS,
			Type:     r.Type,
			Severity: r.Severity,
			Payload:  json.RawMessage(r.PayloadJSON),
		})
	}
	if len(out) > 0 {
		var run store.RunRecord
		if err := s.gdb().First(&run, "id = ?", runID).Error; err == nil {
			for i := range out {
				out[i].TraceID = run.TraceID
			}
		}
	}
	return out, nil
}

func (s *Service) SeqFromEventID(runID, eventID string) (int64, error) {
	var ev store.RunEvent
	if err := s.gdb().Where("run_id = ? AND id = ?", runID, eventID).First(&ev).Error; err != nil {
		return 0, err
	}
	return ev.Seq, nil
}

// BoardLiveEventTypes are Quest kanban-relevant events (DX53).
var BoardLiveEventTypes = []string{
	"plan.created", "plan.approved", "plan.started", "plan.rejected",
	"run.started", "run.finished", "run.failed", "run.canceled",
	"gate.waiting_approval",
}

// TSFromEventID returns the event timestamp for space-stream resume cursors.
func (s *Service) TSFromEventID(eventID string) (int64, error) {
	var ev store.RunEvent
	if err := s.gdb().Where("id = ?", eventID).First(&ev).Error; err != nil {
		return 0, err
	}
	return ev.TS, nil
}

// ListAfterSpace returns board-live events for plan/run ids belonging to spaceID with ts > afterTS.
func (s *Service) ListAfterSpace(spaceID string, afterTS int64, limit int) ([]Envelope, error) {
	if limit <= 0 {
		limit = 100
	}
	spaceID = strings.TrimSpace(spaceID)
	if spaceID == "" {
		spaceID = "local"
	}
	var planIDs []string
	if err := s.gdb().Model(&store.GoalPlan{}).Where("space_id = ?", spaceID).Pluck("id", &planIDs).Error; err != nil {
		return nil, err
	}
	var runIDs []string
	if err := s.gdb().Model(&store.RunRecord{}).Where("space_id = ?", spaceID).Pluck("id", &runIDs).Error; err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(planIDs)+len(runIDs))
	ids = append(ids, planIDs...)
	ids = append(ids, runIDs...)
	if len(ids) == 0 {
		return nil, nil
	}
	var rows []store.RunEvent
	q := s.gdb().Where("run_id IN ? AND ts > ? AND type IN ?", ids, afterTS, BoardLiveEventTypes).
		Order("ts asc, id asc").Limit(limit).Find(&rows)
	if q.Error != nil {
		return nil, q.Error
	}
	out := make([]Envelope, 0, len(rows))
	for _, r := range rows {
		out = append(out, Envelope{
			ID: r.ID, RunID: r.RunID, Seq: r.Seq, TS: r.TS,
			Type: r.Type, Severity: r.Severity, Payload: json.RawMessage(r.PayloadJSON),
		})
	}
	return out, nil
}
