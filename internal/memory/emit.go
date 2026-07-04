package memory

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/ash-repwiki/ash/internal/events"
	"github.com/ash-repwiki/ash/internal/store"
)

var ErrRunNotFound = errors.New("run not found")

// Service implements candidate → review → query (appendix C / G).
type Service struct {
	db     *store.DB
	events *events.Service
	ctx    context.Context
}

func NewService(db *store.DB, ev *events.Service) *Service {
	return &Service{db: db, events: ev}
}

// WithContext returns a shallow copy bound to ctx for Postgres RLS session vars.
func (s *Service) WithContext(ctx context.Context) *Service {
	if s == nil || ctx == nil {
		return s
	}
	return &Service{db: s.db, events: s.events, ctx: ctx}
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

func (s *Service) resolveRun(runID string) (traceID string, err error) {
	if runID == "" {
		return "", nil
	}
	var rec store.RunRecord
	if err := s.gdb().First(&rec, "id = ?", runID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", ErrRunNotFound
		}
		return "", err
	}
	return rec.TraceID, nil
}

func (s *Service) emitRunEvent(runID, traceID, eventType string, payload any) error {
	if s.events == nil || runID == "" {
		return nil
	}
	if traceID == "" {
		var err error
		traceID, err = s.resolveRun(runID)
		if err != nil {
			return err
		}
	}
	_, err := s.events.Append(runID, traceID, eventType, "info", payload)
	return err
}

func (s *Service) requireRunIfSet(runID string) (traceID string, err error) {
	if runID == "" {
		return "", nil
	}
	return s.resolveRun(runID)
}

func (s *Service) emitCandidateCreated(runID, traceID, candidateID, layer, sensitivity string, evidenceCount int) error {
	return s.emitRunEvent(runID, traceID, "memory.candidate_created", map[string]any{
		"candidateId":   candidateID,
		"layer":         layer,
		"evidenceCount": evidenceCount,
		"sensitivity":   sensitivity,
	})
}

func (s *Service) emitReviewed(runID, traceID, candidateID, layer, decision, reason, policy string) error {
	if err := s.emitRunEvent(runID, traceID, "memory.reviewed", map[string]any{
		"candidateId":   candidateID,
		"layer":         layer,
		"decision":      decision,
		"reason":        reason,
		"policyProfile": policy,
	}); err != nil {
		return err
	}
	if decision == "deprecate" {
		return s.emitRunEvent(runID, traceID, "memory.deprecated", map[string]any{
			"memoryId": candidateID,
			"layer":    layer,
			"reason":   reason,
		})
	}
	return nil
}

func (s *Service) emitHitUsed(runID, traceID string, recordIDs []string, hitsByLayer map[string]int) error {
	total := 0
	for _, n := range hitsByLayer {
		total += n
	}
	if total == 0 {
		total = len(recordIDs)
	}
	payload := map[string]any{
		"count":       total,
		"hitsByLayer": hitsByLayer,
	}
	if len(recordIDs) > 0 {
		payload["recordIds"] = recordIDs
	}
	return s.emitRunEvent(runID, traceID, "memory.hit_used", payload)
}

func (s *Service) emitMigrated(runID, traceID string, from, to int, ok bool, recordsUpdated int, summary string) error {
	return s.emitRunEvent(runID, traceID, "memory.migrated", map[string]any{
		"from":           from,
		"to":             to,
		"ok":             ok,
		"recordsUpdated": recordsUpdated,
		"summary":        summary,
		"toolVersion":    MemoryToolVersion,
	})
}

func (s *Service) emitQuery(runID, traceID, layersKey string, resultCount int, latencyMs int64) error {
	return s.emitRunEvent(runID, traceID, "memory.query", map[string]any{
		"layersKey":   layersKey,
		"resultCount": resultCount,
		"latencyMs":   latencyMs,
	})
}

func (s *Service) emitGovernanceEdge(runID, traceID string, edge store.MemoryEdge) error {
	return s.emitRunEvent(runID, traceID, "memory.edge_created", map[string]any{
		"edgeId":     edge.ID,
		"fromId":     edge.FromID,
		"toId":       edge.ToID,
		"kind":       edge.Kind,
		"confidence": edge.Confidence,
		"reason":     edge.Reason,
	})
}

func (s *Service) emitTTLExpired(runID, traceID, memoryID, layer, reason string) error {
	payload := map[string]any{
		"memoryId": memoryID,
		"layer":    layer,
		"reason":   reason,
	}
	if err := s.emitRunEvent(runID, traceID, "memory.ttl_expired", payload); err != nil {
		return err
	}
	return s.emitRunEvent(runID, traceID, "memory.deprecated", payload)
}

func (s *Service) validateRunRef(runID, traceID string) (string, error) {
	resolved, err := s.requireRunIfSet(runID)
	if err != nil {
		return "", fmt.Errorf("%w", err)
	}
	if traceID != "" {
		return traceID, nil
	}
	return resolved, nil
}
