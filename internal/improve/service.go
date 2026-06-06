package improve

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ash-repwiki/ash/internal/artifacts"
	"github.com/ash-repwiki/ash/internal/events"
	"github.com/ash-repwiki/ash/internal/runs"
	"github.com/ash-repwiki/ash/internal/store"
)

var (
	ErrNotFound          = errors.New("proposal not found")
	ErrInvalidState      = errors.New("proposal state does not allow this action")
	ErrBaselineRequired  = errors.New("baseline run is required")
	ErrBaselineNotReady  = errors.New("baseline run is not finished")
)

type Service struct {
	db     *store.DB
	runs   *runs.Service
	events *events.Service
	ctx    context.Context
}

func NewService(db *store.DB, runsSvc *runs.Service, ev *events.Service) *Service {
	return &Service{db: db, runs: runsSvc, events: ev}
}

// WithContext returns a shallow copy bound to ctx for Postgres RLS session vars.
func (s *Service) WithContext(ctx context.Context) *Service {
	if s == nil || ctx == nil {
		return s
	}
	return &Service{
		db: s.db, runs: s.runs.WithContext(ctx), events: s.events.WithContext(ctx), ctx: ctx,
	}
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

func (s *Service) Create(req CreateProposalRequest) (*ProposalView, error) {
	if req.BaselineRunID == "" {
		return nil, ErrBaselineRequired
	}
	var baseline store.RunRecord
	if err := s.gdb().First(&baseline, "id = ?", req.BaselineRunID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, runs.ErrRunNotFound
		}
		return nil, err
	}
	if baseline.Status != "finished" && baseline.Status != "failed" {
		return nil, ErrBaselineNotReady
	}
	now := time.Now().UTC()
	spaceID := firstNonEmpty(req.SpaceID, baseline.SpaceID, "local")
	row := store.ImproveProposal{
		ID:            "imp_" + uuid.NewString(),
		SpaceID:       spaceID,
		Title:         req.Title,
		Description:   req.Description,
		BaselineRunID: req.BaselineRunID,
		Status:        "draft",
		ChangeSummary: req.ChangeSummary,
		ActorID:       req.ActorID,
		CompareJSON:   "{}",
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := s.gdb().Create(&row).Error; err != nil {
		return nil, err
	}
	s.emitProposalEvent(row, "improve.proposal_created", map[string]any{
		"proposalId": row.ID, "baselineRunId": row.BaselineRunID, "title": row.Title,
	})
	return s.view(row), nil
}

func (s *Service) Get(spaceID, id string) (*ProposalView, error) {
	row, err := s.load(spaceID, id)
	if err != nil {
		return nil, err
	}
	return s.view(*row), nil
}

func (s *Service) List(spaceID string, limit int) (*ListProposalsResponse, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	var rows []store.ImproveProposal
	q := s.gdb().Order("updated_at desc").Limit(limit)
	if spaceID != "" {
		q = q.Where("space_id = ?", spaceID)
	}
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]ProposalView, 0, len(rows))
	for _, row := range rows {
		items = append(items, *s.view(row))
	}
	return &ListProposalsResponse{Items: items}, nil
}

func (s *Service) StartExperiment(spaceID, id string) (*StartExperimentResponse, error) {
	row, err := s.load(spaceID, id)
	if err != nil {
		return nil, err
	}
	if row.Status != "draft" && row.Status != "experimenting" {
		return nil, fmt.Errorf("%w: status=%s", ErrInvalidState, row.Status)
	}
	replay, err := s.runs.Replay(row.BaselineRunID, runs.ReplayRequest{Mode: "exact"})
	if err != nil {
		return nil, err
	}
	compare, err := s.compareRuns(row.BaselineRunID, replay.RunID)
	if err != nil {
		return nil, err
	}
	compareJSON, _ := json.Marshal(compare)
	now := time.Now().UTC()
	row.ExperimentRunID = replay.RunID
	row.Status = "experimenting"
	row.CompareJSON = string(compareJSON)
	row.UpdatedAt = now
	if err := s.gdb().Save(row).Error; err != nil {
		return nil, err
	}
	s.emitProposalEvent(*row, "improve.experiment_started", map[string]any{
		"proposalId": row.ID, "baselineRunId": row.BaselineRunID, "experimentRunId": replay.RunID,
	})
	s.emitProposalEvent(*row, "improve.experiment_finished", map[string]any{
		"proposalId": row.ID, "compare": compare,
	})
	return &StartExperimentResponse{
		ProposalID: row.ID, ExperimentRunID: replay.RunID, Compare: compare,
	}, nil
}

func (s *Service) StartCanary(spaceID, id string, req CanaryRequest) (*StatusResponse, error) {
	row, err := s.load(spaceID, id)
	if err != nil {
		return nil, err
	}
	if row.Status != "experimenting" {
		return nil, fmt.Errorf("%w: status=%s", ErrInvalidState, row.Status)
	}
	if req.Percent < 1 || req.Percent > 100 {
		return nil, fmt.Errorf("canary percent must be 1..100")
	}
	row.Status = "canary"
	row.CanaryPercent = req.Percent
	row.UpdatedAt = time.Now().UTC()
	if err := s.gdb().Save(row).Error; err != nil {
		return nil, err
	}
	s.emitProposalEvent(*row, "improve.canary_started", map[string]any{
		"proposalId": row.ID, "percent": req.Percent,
	})
	return &StatusResponse{OK: true, Status: row.Status}, nil
}

func (s *Service) Promote(spaceID, id string) (*StatusResponse, error) {
	row, err := s.load(spaceID, id)
	if err != nil {
		return nil, err
	}
	if row.Status != "canary" && row.Status != "experimenting" {
		return nil, fmt.Errorf("%w: status=%s", ErrInvalidState, row.Status)
	}
	row.Status = "promoted"
	row.UpdatedAt = time.Now().UTC()
	if err := s.gdb().Save(row).Error; err != nil {
		return nil, err
	}
	s.emitProposalEvent(*row, "improve.promoted", map[string]any{"proposalId": row.ID})
	return &StatusResponse{OK: true, Status: row.Status}, nil
}

func (s *Service) Rollback(spaceID, id string) (*StatusResponse, error) {
	row, err := s.load(spaceID, id)
	if err != nil {
		return nil, err
	}
	if row.Status == "rolled_back" || row.Status == "promoted" {
		return nil, fmt.Errorf("%w: status=%s", ErrInvalidState, row.Status)
	}
	row.Status = "rolled_back"
	row.UpdatedAt = time.Now().UTC()
	if err := s.gdb().Save(row).Error; err != nil {
		return nil, err
	}
	s.emitProposalEvent(*row, "improve.rollback", map[string]any{"proposalId": row.ID})
	return &StatusResponse{OK: true, Status: row.Status}, nil
}

func (s *Service) load(spaceID, id string) (*store.ImproveProposal, error) {
	var row store.ImproveProposal
	q := s.gdb().Where("id = ?", id)
	if spaceID != "" {
		q = q.Where("space_id = ?", spaceID)
	}
	if err := q.First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &row, nil
}

func (s *Service) view(row store.ImproveProposal) *ProposalView {
	var compare *ArtifactCompare
	if row.CompareJSON != "" && row.CompareJSON != "{}" {
		var parsed ArtifactCompare
		if json.Unmarshal([]byte(row.CompareJSON), &parsed) == nil {
			compare = &parsed
		}
	}
	return &ProposalView{
		ID: row.ID, Title: row.Title, Description: row.Description,
		BaselineRunID: row.BaselineRunID, ExperimentRunID: row.ExperimentRunID,
		Status: row.Status, ChangeSummary: row.ChangeSummary, CanaryPercent: row.CanaryPercent,
		Compare: compare, CreatedAt: ms(row.CreatedAt), UpdatedAt: ms(row.UpdatedAt),
	}
}

func (s *Service) compareRuns(baselineRunID, experimentRunID string) (*ArtifactCompare, error) {
	base, err := s.runs.Artifacts(baselineRunID)
	if err != nil {
		return nil, err
	}
	exp, err := s.runs.Artifacts(experimentRunID)
	if err != nil {
		return nil, err
	}
	baseByType := digestByType(base)
	expByType := digestByType(exp)
	out := &ArtifactCompare{
		BaselineRunID: baselineRunID, ExperimentRunID: experimentRunID,
		ByType: map[string]string{},
	}
	for typ, digest := range baseByType {
		other, ok := expByType[typ]
		if !ok {
			out.Missing++
			out.ByType[typ] = "missing"
			continue
		}
		if digest == other {
			out.Matched++
			out.ByType[typ] = "match"
		} else {
			out.Changed++
			out.ByType[typ] = "changed"
		}
	}
	for typ := range expByType {
		if _, ok := baseByType[typ]; !ok {
			out.Changed++
			out.ByType[typ] = "new"
		}
	}
	return out, nil
}

func digestByType(m *artifacts.Manifest) map[string]string {
	out := map[string]string{}
	if m == nil {
		return out
	}
	for _, art := range m.Artifacts {
		out[art.Type] = art.Digest
	}
	return out
}

func (s *Service) emitProposalEvent(row store.ImproveProposal, typ string, payload map[string]any) {
	if row.BaselineRunID == "" {
		return
	}
	var rec store.RunRecord
	if err := s.gdb().Select("trace_id").First(&rec, "id = ?", row.BaselineRunID).Error; err != nil {
		return
	}
	_, _ = s.events.Append(row.BaselineRunID, rec.TraceID, typ, "info", payload)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
