package memory

import (
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
}

func NewService(db *store.DB, ev *events.Service) *Service {
	return &Service{db: db, events: ev}
}

func (s *Service) resolveRun(runID string) (traceID string, err error) {
	if runID == "" {
		return "", nil
	}
	var rec store.RunRecord
	if err := s.db.First(&rec, "id = ?", runID).Error; err != nil {
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

func (s *Service) emitReviewed(runID, traceID, candidateID, decision, reason, policy string) error {
	if err := s.emitRunEvent(runID, traceID, "memory.reviewed", map[string]any{
		"candidateId":   candidateID,
		"decision":      decision,
		"reason":        reason,
		"policyProfile": policy,
	}); err != nil {
		return err
	}
	if decision == "deprecate" {
		return s.emitRunEvent(runID, traceID, "memory.deprecated", map[string]any{
			"memoryId": candidateID,
			"reason":   reason,
		})
	}
	return nil
}

func (s *Service) emitHitUsed(runID, traceID string, recordIDs []string) error {
	return s.emitRunEvent(runID, traceID, "memory.hit_used", map[string]any{
		"recordIds": recordIDs,
		"count":     len(recordIDs),
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
