package agentworkspace

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ash-repwiki/ash/internal/store"
)

const (
	StatusActive = "active"
	StatusClosed = "closed"

	auditEventType = "agent.workspace"
	maxTitleRunes  = 64
)

// View is the public agent workspace document (persisted in audit_log).
type View struct {
	ID         string   `json:"id"`
	SpaceID    string   `json:"spaceId"`
	Title      string   `json:"title"`
	RepoRoot   string   `json:"repoRoot,omitempty"`
	SessionIDs []string `json:"sessionIds"`
	CreatedAt  int64    `json:"createdAt"`
	UpdatedAt  int64    `json:"updatedAt"`
	Status     string   `json:"status"` // active|closed
}

// CreateRequest creates a workspace.
type CreateRequest struct {
	Title     string `json:"title"`
	RepoRoot  string `json:"repoRoot"`
	SpaceID   string `json:"spaceId"`
	CreatedBy string `json:"createdBy"`
}

// PatchRequest updates mutable workspace fields.
type PatchRequest struct {
	Title      *string  `json:"title"`
	RepoRoot   *string  `json:"repoRoot"`
	SessionIDs []string `json:"sessionIds"`
}

// AttachRequest appends a session id to a workspace.
type AttachRequest struct {
	SessionID string `json:"sessionId" binding:"required"`
}

// ListResponse is the list envelope for API/swagger.
type ListResponse struct {
	Items []View `json:"items"`
}

// Service persists workspaces as audit_log rows (event_type=agent.workspace).
type Service struct {
	db  *store.DB
	ctx context.Context
}

func NewService(db *store.DB) *Service {
	return &Service{db: db}
}

func (s *Service) WithContext(ctx context.Context) *Service {
	if s == nil || ctx == nil {
		return s
	}
	out := *s
	out.ctx = ctx
	if s.db != nil {
		out.db = s.db.BindContext(ctx)
	}
	return &out
}

func (s *Service) q() *gorm.DB {
	if s.ctx != nil && s.db != nil {
		return s.db.WithContext(s.ctx)
	}
	return s.db.DB
}

// Create starts an active workspace.
func (s *Service) Create(req CreateRequest) (*View, error) {
	space := firstNonEmpty(strings.TrimSpace(req.SpaceID), "local")
	title := truncateTitle(strings.TrimSpace(req.Title), maxTitleRunes)
	if title == "" {
		title = "Workspace"
	}
	now := time.Now().UTC()
	view := &View{
		ID:         "aw_" + uuid.NewString(),
		SpaceID:    space,
		Title:      title,
		RepoRoot:   strings.TrimSpace(req.RepoRoot),
		SessionIDs: []string{},
		CreatedAt:  now.Unix(),
		UpdatedAt:  now.Unix(),
		Status:     StatusActive,
	}
	if err := s.save(view, firstNonEmpty(strings.TrimSpace(req.CreatedBy), "workspace")); err != nil {
		return nil, err
	}
	return view, nil
}

// Get returns a workspace by id.
func (s *Service) Get(workspaceID string) (*View, error) {
	row, err := s.loadRow(workspaceID)
	if err != nil {
		return nil, err
	}
	return decodeView(row)
}

// List returns active workspaces for a space (newest UpdatedAt first).
func (s *Service) List(spaceID string, limit int) ([]View, error) {
	spaceID = firstNonEmpty(strings.TrimSpace(spaceID), "local")
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	var rows []store.AuditLog
	if err := s.q().Where("event_type = ? AND space_id = ?", auditEventType, spaceID).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]View, 0, len(rows))
	for _, row := range rows {
		view, err := decodeView(row)
		if err != nil {
			continue
		}
		if view.Status == StatusClosed {
			continue
		}
		if view.CreatedAt == 0 && !row.CreatedAt.IsZero() {
			view.CreatedAt = row.CreatedAt.Unix()
		}
		if view.UpdatedAt == 0 {
			view.UpdatedAt = view.CreatedAt
		}
		out = append(out, *view)
	}
	sort.Slice(out, func(i, j int) bool {
		ai := out[i].UpdatedAt
		if ai == 0 {
			ai = out[i].CreatedAt
		}
		aj := out[j].UpdatedAt
		if aj == 0 {
			aj = out[j].CreatedAt
		}
		if ai != aj {
			return ai > aj
		}
		return out[i].ID > out[j].ID
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// Patch renames, updates repoRoot, and/or replaces sessionIds.
func (s *Service) Patch(workspaceID string, req PatchRequest) (*View, error) {
	view, err := s.Get(workspaceID)
	if err != nil {
		return nil, err
	}
	if view.Status == StatusClosed {
		return nil, fmt.Errorf("workspace status %q cannot be patched", view.Status)
	}
	if req.Title != nil {
		title := truncateTitle(strings.TrimSpace(*req.Title), maxTitleRunes)
		if title == "" {
			title = "Workspace"
		}
		view.Title = title
	}
	if req.RepoRoot != nil {
		view.RepoRoot = strings.TrimSpace(*req.RepoRoot)
	}
	if req.SessionIDs != nil {
		view.SessionIDs = normalizeSessionIDs(req.SessionIDs)
	}
	view.UpdatedAt = time.Now().UTC().Unix()
	if err := s.save(view, "workspace"); err != nil {
		return nil, err
	}
	return view, nil
}

// Attach appends sessionID if missing.
func (s *Service) Attach(workspaceID, sessionID string) (*View, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil, fmt.Errorf("sessionId is required")
	}
	view, err := s.Get(workspaceID)
	if err != nil {
		return nil, err
	}
	if view.Status != StatusActive {
		return nil, fmt.Errorf("workspace status %q cannot accept sessions", view.Status)
	}
	for _, id := range view.SessionIDs {
		if id == sessionID {
			return view, nil
		}
	}
	view.SessionIDs = append(view.SessionIDs, sessionID)
	view.UpdatedAt = time.Now().UTC().Unix()
	if err := s.save(view, "workspace"); err != nil {
		return nil, err
	}
	return view, nil
}

// Close soft-closes a workspace (status=closed). List excludes closed.
func (s *Service) Close(workspaceID string) (*View, error) {
	view, err := s.Get(workspaceID)
	if err != nil {
		return nil, err
	}
	if view.Status == StatusClosed {
		return view, nil
	}
	view.Status = StatusClosed
	view.UpdatedAt = time.Now().UTC().Unix()
	if err := s.save(view, "workspace"); err != nil {
		return nil, err
	}
	return view, nil
}

// FindIDBySession returns the newest active workspace id that lists sessionID.
func (s *Service) FindIDBySession(spaceID, sessionID string) (string, bool) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return "", false
	}
	items, err := s.List(spaceID, 200)
	if err != nil {
		return "", false
	}
	for _, item := range items {
		for _, id := range item.SessionIDs {
			if id == sessionID {
				return item.ID, true
			}
		}
	}
	return "", false
}

func (s *Service) loadRow(workspaceID string) (store.AuditLog, error) {
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return store.AuditLog{}, fmt.Errorf("workspaceId is required")
	}
	var row store.AuditLog
	if err := s.q().First(&row, "id = ? AND event_type = ?", workspaceID, auditEventType).Error; err != nil {
		return store.AuditLog{}, fmt.Errorf("workspace not found: %w", err)
	}
	return row, nil
}

func (s *Service) save(view *View, actorID string) error {
	if view == nil {
		return fmt.Errorf("workspace view is nil")
	}
	if view.SessionIDs == nil {
		view.SessionIDs = []string{}
	}
	payload, err := json.Marshal(view)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	row := store.AuditLog{
		ID: view.ID, SpaceID: view.SpaceID,
		ActorID: firstNonEmpty(actorID, "workspace"), EventType: auditEventType,
		PayloadJSON: string(payload), CreatedAt: time.Unix(view.CreatedAt, 0).UTC(),
	}
	if row.CreatedAt.IsZero() {
		row.CreatedAt = now
	}
	var existing store.AuditLog
	err = s.q().First(&existing, "id = ?", view.ID).Error
	if err == gorm.ErrRecordNotFound {
		return s.q().Create(&row).Error
	}
	if err != nil {
		return err
	}
	return s.q().Model(&store.AuditLog{}).Where("id = ?", view.ID).Updates(map[string]any{
		"space_id": view.SpaceID, "actor_id": row.ActorID, "payload_json": string(payload),
	}).Error
}

func decodeView(row store.AuditLog) (*View, error) {
	var view View
	if err := json.Unmarshal([]byte(row.PayloadJSON), &view); err != nil {
		return nil, fmt.Errorf("decode workspace: %w", err)
	}
	if view.ID == "" {
		view.ID = row.ID
	}
	if view.SessionIDs == nil {
		view.SessionIDs = []string{}
	}
	if view.Status == "" {
		view.Status = StatusActive
	}
	return &view, nil
}

func normalizeSessionIDs(ids []string) []string {
	seen := make(map[string]struct{}, len(ids))
	out := make([]string, 0, len(ids))
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

func truncateTitle(text string, maxRunes int) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	if i := strings.IndexByte(text, '\n'); i >= 0 {
		text = strings.TrimSpace(text[:i])
	}
	if maxRunes <= 0 {
		return text
	}
	runes := []rune(text)
	if len(runes) <= maxRunes {
		return string(runes)
	}
	return string(runes[:maxRunes])
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
