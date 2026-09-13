package registry

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ash-repwiki/ash/internal/store"
)

type Service struct {
	db *store.DB
}

func NewService(db *store.DB) *Service {
	return &Service{db: db}
}

func (s *Service) WithDB(db *store.DB) *Service {
	if s == nil {
		return nil
	}
	out := *s
	if db != nil {
		out.db = db
	}
	return &out
}

func (s *Service) q() *gorm.DB {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.DB
}

func (s *Service) CreateAgentAsset(req CreateAgentAssetRequest) (*AgentAssetView, error) {
	kind := strings.ToLower(strings.TrimSpace(req.Kind))
	if _, ok := AgentKinds[kind]; !ok {
		return nil, fmt.Errorf("kind must be harness_profile|adapter|skill|tool|plugin")
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	status := strings.ToLower(strings.TrimSpace(req.Status))
	if status == "" {
		status = StatusDraft
	}
	if _, ok := AssetStatuses[status]; !ok {
		return nil, fmt.Errorf("status must be draft|active|disabled")
	}
	space := firstNonEmpty(strings.TrimSpace(req.SpaceID), "local")
	now := time.Now().UTC()
	row := store.AgentAsset{
		ID: "aasset_" + uuid.NewString(), SpaceID: space, Kind: kind,
		RefID: strings.TrimSpace(req.RefID), Name: name, Status: status,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := s.q().Create(&row).Error; err != nil {
		return nil, err
	}
	return agentView(row), nil
}

func (s *Service) ListAgentAssets(spaceID, status string, limit int) (*AgentAssetListResponse, error) {
	space := firstNonEmpty(strings.TrimSpace(spaceID), "local")
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	q := s.q().Where("space_id = ?", space).Order("created_at desc").Limit(limit)
	if st := strings.ToLower(strings.TrimSpace(status)); st != "" {
		q = q.Where("status = ?", st)
	}
	var rows []store.AgentAsset
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]AgentAssetView, 0, len(rows))
	for _, r := range rows {
		out = append(out, *agentView(r))
	}
	return &AgentAssetListResponse{Items: out}, nil
}

func (s *Service) PatchAgentAssetStatus(id, status string) (*AgentAssetView, error) {
	id = strings.TrimSpace(id)
	status = strings.ToLower(strings.TrimSpace(status))
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	if _, ok := AssetStatuses[status]; !ok {
		return nil, fmt.Errorf("status must be draft|active|disabled")
	}
	var row store.AgentAsset
	if err := s.q().First(&row, "id = ?", id).Error; err != nil {
		return nil, fmt.Errorf("agent asset not found: %w", err)
	}
	row.Status = status
	row.UpdatedAt = time.Now().UTC()
	if err := s.q().Save(&row).Error; err != nil {
		return nil, err
	}
	return agentView(row), nil
}

func (s *Service) CreateMemoryAsset(req CreateMemoryAssetRequest) (*MemoryAssetView, error) {
	kind := strings.ToLower(strings.TrimSpace(req.Kind))
	if _, ok := MemoryKinds[kind]; !ok {
		return nil, fmt.Errorf("kind must be record|skill|pack|edge")
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	status := strings.ToLower(strings.TrimSpace(req.Status))
	if status == "" {
		status = StatusDraft
	}
	if _, ok := AssetStatuses[status]; !ok {
		return nil, fmt.Errorf("status must be draft|active|disabled")
	}
	space := firstNonEmpty(strings.TrimSpace(req.SpaceID), "local")
	now := time.Now().UTC()
	row := store.MemoryAsset{
		ID: "masset_" + uuid.NewString(), SpaceID: space, Kind: kind,
		RefID: strings.TrimSpace(req.RefID), Name: name, Status: status,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := s.q().Create(&row).Error; err != nil {
		return nil, err
	}
	return memoryView(row), nil
}

func (s *Service) ListMemoryAssets(spaceID, status string, limit int) (*MemoryAssetListResponse, error) {
	space := firstNonEmpty(strings.TrimSpace(spaceID), "local")
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	q := s.q().Where("space_id = ?", space).Order("created_at desc").Limit(limit)
	if st := strings.ToLower(strings.TrimSpace(status)); st != "" {
		q = q.Where("status = ?", st)
	}
	var rows []store.MemoryAsset
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]MemoryAssetView, 0, len(rows))
	for _, r := range rows {
		out = append(out, *memoryView(r))
	}
	return &MemoryAssetListResponse{Items: out}, nil
}

func (s *Service) PatchMemoryAssetStatus(id, status string) (*MemoryAssetView, error) {
	id = strings.TrimSpace(id)
	status = strings.ToLower(strings.TrimSpace(status))
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	if _, ok := AssetStatuses[status]; !ok {
		return nil, fmt.Errorf("status must be draft|active|disabled")
	}
	var row store.MemoryAsset
	if err := s.q().First(&row, "id = ?", id).Error; err != nil {
		return nil, fmt.Errorf("memory asset not found: %w", err)
	}
	row.Status = status
	row.UpdatedAt = time.Now().UTC()
	if err := s.q().Save(&row).Error; err != nil {
		return nil, err
	}
	return memoryView(row), nil
}

func agentView(r store.AgentAsset) *AgentAssetView {
	return &AgentAssetView{
		ID: r.ID, SpaceID: r.SpaceID, Kind: r.Kind, RefID: r.RefID,
		Name: r.Name, Status: r.Status, CreatedAt: ms(r.CreatedAt),
	}
}

func memoryView(r store.MemoryAsset) *MemoryAssetView {
	return &MemoryAssetView{
		ID: r.ID, SpaceID: r.SpaceID, Kind: r.Kind, RefID: r.RefID,
		Name: r.Name, Status: r.Status, CreatedAt: ms(r.CreatedAt),
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
