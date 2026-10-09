package memory

import (
	"fmt"
	"strings"
	"time"

	"github.com/ash-repwiki/ash/internal/store"
)

// ForgetRequest proposes deprecating an approved memory without applying it.
type ForgetRequest struct {
	SpaceID  string
	MemoryID string
	ActorID  string
	RunID    string
	TraceID  string
	Reason   string
}

// ForgetResponse is the review-queue candidate for a forget proposal.
type ForgetResponse struct {
	CandidateID string
	MemoryID    string
}

// ForgetPropose creates a forget candidate. It does not deprecate the target until review.
func (s *Service) ForgetPropose(req ForgetRequest) (*ForgetResponse, error) {
	memoryID := strings.TrimSpace(req.MemoryID)
	if memoryID == "" {
		return nil, fmt.Errorf("memoryId is required")
	}
	spaceID := firstNonEmpty(req.SpaceID, "local")
	target, err := s.requireApproved(spaceID, memoryID)
	if err != nil {
		return nil, err
	}
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		reason = forgetReason(target)
	}
	created, err := s.CreateCandidate(CreateCandidateRequest{
		Layer: "L0", Title: "forget:" + target.Title, Body: reason,
		SpaceID: spaceID, ActorID: req.ActorID, RunID: req.RunID, TraceID: req.TraceID,
		Tags: []string{"mem.forget"}, Evidence: []EvidenceInput{{Kind: "url", Ref: "memory:" + memoryID}},
	})
	if err != nil {
		return nil, err
	}
	_ = s.emitRunEvent(req.RunID, req.TraceID, "memory.forget_proposed", map[string]any{
		"component": "mem.forget", "candidateId": created.CandidateID, "memoryId": memoryID, "reason": reason,
	})
	var still store.MemoryRecord
	if err := s.gdb().Where("id = ?", memoryID).First(&still).Error; err != nil {
		return nil, err
	}
	if still.Status != "approved" {
		return nil, fmt.Errorf("forget must leave target approved until review")
	}
	return &ForgetResponse{CandidateID: created.CandidateID, MemoryID: memoryID}, nil
}

func forgetReason(rec *store.MemoryRecord) string {
	if rec.TTLDays != nil && *rec.TTLDays >= 0 {
		age := time.Since(rec.UpdatedAt)
		if age > time.Duration(*rec.TTLDays)*24*time.Hour {
			return "ttl expired"
		}
	}
	if rec.Confidence > 0 && rec.Confidence < 0.4 {
		return "low confidence"
	}
	return "long unused or curator request"
}
