package scenariopatch

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/ash-repwiki/ash/internal/store"
)

const (
	StatusDraft    = "draft"
	StatusInReview = "in_review"
	StatusApproved = "approved"
	StatusRejected = "rejected"
	StatusArchived = "archived"
)

type View struct {
	ID           string `json:"id"`
	SpaceID      string `json:"spaceId"`
	ScenarioName string `json:"scenarioName"`
	FromVersion  string `json:"fromVersion,omitempty"`
	ToVersion    string `json:"toVersion,omitempty"`
	Title        string `json:"title"`
	DiffText     string `json:"diffText"`
	Status       string `json:"status"`
	CreatedBy    string `json:"createdBy,omitempty"`
	DecidedBy    string `json:"decidedBy,omitempty"`
	DecisionNote string `json:"decisionNote,omitempty"`
	CreatedAt    int64  `json:"createdAt"`
	UpdatedAt    int64  `json:"updatedAt"`
	DecidedAt    *int64 `json:"decidedAt,omitempty"`
}

type CreateRequest struct {
	SpaceID      string `json:"spaceId"`
	ScenarioName string `json:"scenarioName"`
	FromVersion  string `json:"fromVersion"`
	ToVersion    string `json:"toVersion"`
	Title        string `json:"title"`
	DiffText     string `json:"diffText"`
	CreatedBy    string `json:"createdBy"`
}

type Service struct {
	db *store.DB
}

func NewService(db *store.DB) *Service { return &Service{db: db} }

func (s *Service) WithContext(ctx context.Context) *Service {
	if s == nil || ctx == nil {
		return s
	}
	return &Service{db: s.db.BindContext(ctx)}
}

func (s *Service) Create(req CreateRequest) (*View, error) {
	space := strings.TrimSpace(req.SpaceID)
	if space == "" {
		space = "local"
	}
	name := strings.TrimSpace(req.ScenarioName)
	title := strings.TrimSpace(req.Title)
	diff := strings.TrimSpace(req.DiffText)
	if name == "" || title == "" || diff == "" {
		return nil, fmt.Errorf("scenarioName, title, and diffText are required")
	}
	now := time.Now().UTC()
	row := store.ScenarioPatchDraft{
		ID: "spatch_" + uuid.NewString(), SpaceID: space, ScenarioName: name,
		FromVersion: strings.TrimSpace(req.FromVersion), ToVersion: strings.TrimSpace(req.ToVersion),
		Title: title, DiffText: diff, Status: StatusDraft, CreatedBy: strings.TrimSpace(req.CreatedBy),
		CreatedAt: now, UpdatedAt: now,
	}
	if err := s.db.Create(&row).Error; err != nil {
		return nil, err
	}
	return toView(row), nil
}

func (s *Service) List(spaceID, status string) ([]View, error) {
	space := strings.TrimSpace(spaceID)
	if space == "" {
		space = "local"
	}
	q := s.db.Where("space_id = ?", space)
	if st := strings.TrimSpace(status); st != "" {
		q = q.Where("status = ?", st)
	}
	var rows []store.ScenarioPatchDraft
	if err := q.Order("updated_at desc").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]View, 0, len(rows))
	for _, row := range rows {
		out = append(out, *toView(row))
	}
	return out, nil
}

func (s *Service) Get(id string) (*View, error) {
	var row store.ScenarioPatchDraft
	if err := s.db.First(&row, "id = ?", strings.TrimSpace(id)).Error; err != nil {
		return nil, err
	}
	return toView(row), nil
}

func (s *Service) SubmitReview(id string) (*View, error) {
	var row store.ScenarioPatchDraft
	if err := s.db.First(&row, "id = ?", strings.TrimSpace(id)).Error; err != nil {
		return nil, err
	}
	if row.Status != StatusDraft {
		return nil, fmt.Errorf("only draft patches can be submitted")
	}
	row.Status = StatusInReview
	row.UpdatedAt = time.Now().UTC()
	if err := s.db.Save(&row).Error; err != nil {
		return nil, err
	}
	return toView(row), nil
}

func (s *Service) Decide(id, decision, actorID, note string) (*View, error) {
	decision = strings.ToLower(strings.TrimSpace(decision))
	var row store.ScenarioPatchDraft
	if err := s.db.First(&row, "id = ?", strings.TrimSpace(id)).Error; err != nil {
		return nil, err
	}
	if row.Status != StatusInReview {
		return nil, fmt.Errorf("only in_review patches can be decided")
	}
	now := time.Now().UTC()
	switch decision {
	case "approve":
		row.Status = StatusApproved
	case "reject":
		row.Status = StatusRejected
	default:
		return nil, fmt.Errorf("decision must be approve|reject")
	}
	row.DecidedBy = strings.TrimSpace(actorID)
	row.DecisionNote = strings.TrimSpace(note)
	row.DecidedAt = &now
	row.UpdatedAt = now
	if err := s.db.Save(&row).Error; err != nil {
		return nil, err
	}
	return toView(row), nil
}

func toView(row store.ScenarioPatchDraft) *View {
	v := &View{
		ID: row.ID, SpaceID: row.SpaceID, ScenarioName: row.ScenarioName,
		FromVersion: row.FromVersion, ToVersion: row.ToVersion, Title: row.Title,
		DiffText: row.DiffText, Status: row.Status, CreatedBy: row.CreatedBy,
		DecidedBy: row.DecidedBy, DecisionNote: row.DecisionNote,
		CreatedAt: row.CreatedAt.UTC().UnixMilli(), UpdatedAt: row.UpdatedAt.UTC().UnixMilli(),
	}
	if row.DecidedAt != nil {
		ms := row.DecidedAt.UTC().UnixMilli()
		v.DecidedAt = &ms
	}
	return v
}
