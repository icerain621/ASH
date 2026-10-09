package memory

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ash-repwiki/ash/internal/store"
)

// ConsolidateRequest proposes merging duplicates into a primary record.
type ConsolidateRequest struct {
	SpaceID      string
	PrimaryID    string
	DuplicateIDs []string
	ActorID      string
	RunID        string
	TraceID      string
	Reason       string
}

// ConsolidateResponse is the review-queue candidate for a consolidate proposal.
type ConsolidateResponse struct {
	CandidateID  string
	PrimaryID    string
	DuplicateIDs []string
}

// Consolidate creates a candidate proposal and consolidate edges. It never rewrites approved bodies.
func (s *Service) Consolidate(req ConsolidateRequest) (*ConsolidateResponse, error) {
	primaryID := strings.TrimSpace(req.PrimaryID)
	if primaryID == "" {
		return nil, fmt.Errorf("primaryId is required")
	}
	dups := uniqueNonEmpty(req.DuplicateIDs)
	if len(dups) == 0 {
		return nil, fmt.Errorf("at least one duplicateId is required")
	}
	for _, id := range dups {
		if id == primaryID {
			return nil, fmt.Errorf("duplicateId must not equal primaryId")
		}
	}
	spaceID := firstNonEmpty(req.SpaceID, "local")
	primary, err := s.requireApproved(spaceID, primaryID)
	if err != nil {
		return nil, err
	}
	for _, id := range dups {
		if _, err := s.requireApproved(spaceID, id); err != nil {
			return nil, err
		}
	}
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		reason = "consolidate proposal"
	}
	evidence := []EvidenceInput{{Kind: "url", Ref: "memory:" + primaryID}}
	for _, id := range dups {
		evidence = append(evidence, EvidenceInput{Kind: "url", Ref: "memory:" + id})
	}
	created, err := s.CreateCandidate(CreateCandidateRequest{
		Layer: "L0", Title: "consolidate:" + primary.Title, Body: reason,
		SpaceID: spaceID, ActorID: req.ActorID, RunID: req.RunID, TraceID: req.TraceID,
		Tags: []string{"mem.consolidate"}, Evidence: evidence,
	})
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	err = s.gdb().Transaction(func(tx *gorm.DB) error {
		for _, dupID := range dups {
			edge := store.MemoryEdge{
				ID: "medge_" + uuid.NewString(), SpaceID: spaceID,
				FromID: created.CandidateID, ToID: dupID, Kind: "consolidate_proposal",
				Confidence: 1, Reason: reason, CreatedAt: now,
			}
			if err := tx.Create(&edge).Error; err != nil {
				return err
			}
		}
		edge := store.MemoryEdge{
			ID: "medge_" + uuid.NewString(), SpaceID: spaceID,
			FromID: created.CandidateID, ToID: primaryID, Kind: "consolidate_primary",
			Confidence: 1, Reason: reason, CreatedAt: now,
		}
		return tx.Create(&edge).Error
	})
	if err != nil {
		return nil, err
	}
	_ = s.emitRunEvent(req.RunID, req.TraceID, "memory.consolidate_proposed", map[string]any{
		"component": "mem.consolidate", "candidateId": created.CandidateID,
		"primaryId": primaryID, "duplicateIds": dups,
	})
	// Prove approved body unchanged.
	after, err := s.requireApproved(spaceID, primaryID)
	if err != nil {
		return nil, err
	}
	if after.Body != primary.Body || after.Title != primary.Title {
		return nil, fmt.Errorf("consolidate must not mutate approved body")
	}
	return &ConsolidateResponse{CandidateID: created.CandidateID, PrimaryID: primaryID, DuplicateIDs: dups}, nil
}

func (s *Service) requireApproved(spaceID, id string) (*store.MemoryRecord, error) {
	var rec store.MemoryRecord
	if err := s.gdb().Where("id = ? AND space_id = ?", id, spaceID).First(&rec).Error; err != nil {
		return nil, fmt.Errorf("memory %s not found", id)
	}
	if rec.Status != "approved" {
		return nil, fmt.Errorf("memory %s status %q want approved", id, rec.Status)
	}
	return &rec, nil
}

func uniqueNonEmpty(ids []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}
