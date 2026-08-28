package quest

import (
	"context"
	"sort"

	"github.com/ash-repwiki/ash/internal/goal"
	"github.com/ash-repwiki/ash/internal/runs"
	"github.com/ash-repwiki/ash/internal/store"
)

type BoardItem struct {
	ID       string `json:"id"`
	Kind     string `json:"kind"` // plan|run
	Title    string `json:"title"`
	Status   string `json:"status"`
	Column   string `json:"column"`
	Scenario string `json:"scenario,omitempty"`
	RunID    string `json:"runId,omitempty"`
	PlanID   string `json:"planId,omitempty"`
	SpaceID  string `json:"spaceId"`
	UpdatedAt int64 `json:"updatedAt"`
}

type BoardResponse struct {
	Columns map[string][]BoardItem `json:"columns"`
}

type Service struct {
	db   *store.DB
	runs *runs.Service
}

func NewService(db *store.DB, runsSvc *runs.Service) *Service {
	return &Service{db: db, runs: runsSvc}
}

func (s *Service) WithContext(ctx context.Context) *Service {
	if s == nil || ctx == nil {
		return s
	}
	out := *s
	out.db = s.db.BindContext(ctx)
	if s.runs != nil {
		out.runs = s.runs.WithContext(ctx)
	}
	return &out
}

func (s *Service) Board(spaceID string, limit int) (*BoardResponse, error) {
	if limit <= 0 {
		limit = 80
	}
	cols := map[string][]BoardItem{
		"plans":            {},
		"running":          {},
		"waiting_approval": {},
		"finished":         {},
	}

	var plans []store.GoalPlan
	_ = s.db.Where("space_id = ?", spaceID).Order("updated_at desc").Limit(limit).Find(&plans).Error
	for _, p := range plans {
		col := "plans"
		if p.Status == goal.StatusStarted && p.RunID != "" {
			continue // run card will represent it
		}
		if p.Status == goal.StatusRejected {
			col = "finished"
		}
		cols[col] = append(cols[col], BoardItem{
			ID: p.ID, Kind: "plan", Title: truncate(p.Goal, 80), Status: p.Status, Column: col,
			Scenario: p.ScenarioName + "@" + p.ScenarioVersion, PlanID: p.ID, SpaceID: p.SpaceID,
			UpdatedAt: p.UpdatedAt.UnixMilli(),
		})
	}

	runItems, err := s.runs.ListForSpace(spaceID, limit)
	if err != nil {
		return nil, err
	}
	for _, r := range runItems {
		col := columnForRun(r.Status)
		cols[col] = append(cols[col], BoardItem{
			ID: r.RunID, Kind: "run", Title: r.Scenario.Name + "@" + r.Scenario.ScenarioVersion,
			Status: r.Status, Column: col, Scenario: r.Scenario.Name, RunID: r.RunID,
			SpaceID: r.SpaceID, UpdatedAt: r.StartedAt,
		})
	}
	for k := range cols {
		sort.Slice(cols[k], func(i, j int) bool { return cols[k][i].UpdatedAt > cols[k][j].UpdatedAt })
	}
	return &BoardResponse{Columns: cols}, nil
}

func columnForRun(status string) string {
	switch status {
	case runs.StatusWaitingApproval:
		return "waiting_approval"
	case runs.StatusFinished, runs.StatusFailed, runs.StatusCanceled:
		return "finished"
	default:
		return "running"
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
