package diffreview

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/ash-repwiki/ash/internal/diffparse"
	"github.com/ash-repwiki/ash/internal/runs"
	"github.com/ash-repwiki/ash/internal/store"
)

type DiffView struct {
	RunID         string               `json:"runId"`
	Raw           string               `json:"raw"`
	Files         []diffparse.FileDiff `json:"files"`
	ContextRefs   []string             `json:"contextRefs,omitempty"`
	RejectedPaths []string             `json:"rejectedPaths,omitempty"`
}

type CommentView struct {
	ID        string `json:"id"`
	SpaceID   string `json:"spaceId"`
	RunID     string `json:"runId"`
	FilePath  string `json:"filePath"`
	LineIndex int    `json:"lineIndex"`
	Side      string `json:"side"`
	Body      string `json:"body"`
	CreatedBy string `json:"createdBy,omitempty"`
	CreatedAt int64  `json:"createdAt"`
}

type CreateCommentRequest struct {
	FilePath  string `json:"filePath"`
	LineIndex int    `json:"lineIndex"`
	Side      string `json:"side"`
	Body      string `json:"body"`
	CreatedBy string `json:"createdBy"`
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

func (s *Service) GetDiff(runID string) (*DiffView, error) {
	sum, err := s.runs.Get(runID)
	if err != nil {
		return nil, err
	}
	_ = sum
	runDir := s.db.RunDir(runID)
	path := filepath.Join(runDir, "artifacts", "diff.patch")
	body, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			view := &DiffView{RunID: runID, Raw: "", Files: nil}
			if paths, err := s.RejectedPaths(runID); err == nil {
				view.RejectedPaths = paths
			}
			return view, nil
		}
		return nil, err
	}
	raw := string(body)
	view := &DiffView{RunID: runID, Raw: raw, Files: diffparse.ParseUnified(raw)}
	if man, err := s.runs.Artifacts(runID); err == nil && man != nil {
		view.ContextRefs = man.ContextRefs
	}
	if paths, err := s.RejectedPaths(runID); err == nil {
		view.RejectedPaths = paths
	}
	return view, nil
}

func (s *Service) ListComments(runID string) ([]CommentView, error) {
	if _, err := s.runs.Get(runID); err != nil {
		return nil, err
	}
	var rows []store.DiffReviewComment
	if err := s.db.Where("run_id = ?", runID).Order("created_at asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]CommentView, 0, len(rows))
	for _, r := range rows {
		out = append(out, toComment(r))
	}
	return out, nil
}

func (s *Service) CreateComment(runID string, req CreateCommentRequest) (*CommentView, error) {
	sum, err := s.runs.Get(runID)
	if err != nil {
		return nil, err
	}
	body := strings.TrimSpace(req.Body)
	path := strings.TrimSpace(req.FilePath)
	if body == "" || path == "" {
		return nil, fmt.Errorf("filePath and body are required")
	}
	side := strings.ToLower(strings.TrimSpace(req.Side))
	if side == "" {
		side = "new"
	}
	now := time.Now().UTC()
	row := store.DiffReviewComment{
		ID: "drc_" + uuid.NewString(), SpaceID: sum.SpaceID, RunID: runID,
		FilePath: path, LineIndex: req.LineIndex, Side: side, Body: body,
		CreatedBy: strings.TrimSpace(req.CreatedBy), CreatedAt: now, UpdatedAt: now,
	}
	if err := s.db.Create(&row).Error; err != nil {
		return nil, err
	}
	v := toComment(row)
	return &v, nil
}

const (
	RejectScopeFile = "file"
	RejectScopeAll  = "all"
	RejectSide      = "reject"
	RejectAllPath   = "*"
	RejectLineIndex = -1
)

type RejectRequest struct {
	Scope    string `json:"scope"` // file|all
	FilePath string `json:"filePath,omitempty"`
	Reason   string `json:"reason"`
	ActorID  string `json:"actorId,omitempty"`
}

type RejectResponse struct {
	RunID    string      `json:"runId"`
	Scope    string      `json:"scope"`
	FilePath string      `json:"filePath"`
	Comment  CommentView `json:"comment"`
	Canceled bool        `json:"canceled"`
	Status   string      `json:"status"`
}

// Reject records a file-level or full-diff rejection (no new tables).
// scope=all also cancels the run so delivery stops; scope=file records only.
func (s *Service) Reject(runID string, req RejectRequest) (*RejectResponse, error) {
	sum, err := s.runs.Get(runID)
	if err != nil {
		return nil, err
	}
	scope := strings.ToLower(strings.TrimSpace(req.Scope))
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		reason = "diff rejected"
	}
	path := strings.TrimSpace(req.FilePath)
	switch scope {
	case RejectScopeFile:
		if path == "" || path == RejectAllPath {
			return nil, fmt.Errorf("filePath is required for scope=file")
		}
	case RejectScopeAll:
		path = RejectAllPath
	default:
		return nil, fmt.Errorf("scope must be file or all")
	}

	now := time.Now().UTC()
	row := store.DiffReviewComment{
		ID: "drc_" + uuid.NewString(), SpaceID: sum.SpaceID, RunID: runID,
		FilePath: path, LineIndex: RejectLineIndex, Side: RejectSide, Body: reason,
		CreatedBy: strings.TrimSpace(req.ActorID), CreatedAt: now, UpdatedAt: now,
	}
	if err := s.db.Create(&row).Error; err != nil {
		return nil, err
	}
	comment := toComment(row)
	out := &RejectResponse{
		RunID: runID, Scope: scope, FilePath: path, Comment: comment, Status: sum.Status,
	}
	if scope == RejectScopeAll {
		canceled, err := s.runs.Cancel(runID)
		if err != nil {
			return nil, err
		}
		out.Canceled = true
		out.Status = canceled.Status
	}
	if s.runs != nil {
		_ = s.runs.AppendEvent(runID, sum.TraceID, "quest.diff.rejected", "warn", map[string]any{
			"scope": scope, "filePath": path, "reason": reason, "actorId": req.ActorID,
			"canceled": out.Canceled,
		})
	}
	return out, nil
}

// RejectedPaths returns file paths marked rejected (side=reject); "*" means full reject.
func (s *Service) RejectedPaths(runID string) ([]string, error) {
	items, err := s.ListComments(runID)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var out []string
	for _, c := range items {
		if c.Side != RejectSide {
			continue
		}
		p := c.FilePath
		if p == "" {
			p = RejectAllPath
		}
		if seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}
	return out, nil
}

func toComment(r store.DiffReviewComment) CommentView {
	return CommentView{
		ID: r.ID, SpaceID: r.SpaceID, RunID: r.RunID, FilePath: r.FilePath,
		LineIndex: r.LineIndex, Side: r.Side, Body: r.Body, CreatedBy: r.CreatedBy,
		CreatedAt: r.CreatedAt.UnixMilli(),
	}
}
