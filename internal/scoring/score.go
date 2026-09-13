package scoring

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/ash-repwiki/ash/internal/store"
)

// ScoreEventView is the persisted score event DTO.
type ScoreEventView struct {
	ID          string       `json:"id"`
	SpaceID     string       `json:"spaceId"`
	TargetType  string       `json:"targetType"`
	TargetID    string       `json:"targetId"`
	RunID       string       `json:"runId,omitempty"`
	Rubric      ReviewRubric `json:"rubric"`
	Composite   float64      `json:"composite"`
	ActorID     string       `json:"actorId,omitempty"`
	Reason      string       `json:"reason,omitempty"`
	CreatedAt   int64        `json:"createdAt"`
}

// RecordScore persists a validated rubric score event.
func (s *Service) RecordScore(spaceID, targetType, targetID, runID string, rubric ReviewRubric, actor, reason string) (*ScoreEventView, error) {
	if err := rubric.Validate(); err != nil {
		return nil, err
	}
	tt := strings.ToLower(strings.TrimSpace(targetType))
	tid := strings.TrimSpace(targetID)
	if tt == "" || tid == "" {
		return nil, fmt.Errorf("targetType and targetId are required")
	}
	space := strings.TrimSpace(spaceID)
	if space == "" {
		space = "local"
	}
	comp := rubric.Composite()
	now := time.Now().UTC()
	row := store.ScoreEvent{
		ID: "score_" + uuid.NewString(), SpaceID: space,
		TargetType: tt, TargetID: tid, RunID: strings.TrimSpace(runID),
		Correctness: rubric.Correctness, Safety: rubric.Safety,
		Citable: rubric.Citable, Efficiency: rubric.Efficiency,
		Composite: comp, ActorID: strings.TrimSpace(actor),
		Reason: strings.TrimSpace(reason), CreatedAt: now,
	}
	if err := s.gdb().Create(&row).Error; err != nil {
		return nil, err
	}
	return &ScoreEventView{
		ID: row.ID, SpaceID: row.SpaceID, TargetType: row.TargetType, TargetID: row.TargetID,
		RunID: row.RunID, Rubric: rubric, Composite: comp, ActorID: row.ActorID,
		Reason: row.Reason, CreatedAt: now.UnixMilli(),
	}, nil
}
