package harness

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"gorm.io/gorm"

	"github.com/ash-repwiki/ash/internal/store"
)

//go:embed schema/ash.harness.profile.v1.json
var profileSchemaJSON []byte

const (
	StatusDraft    = "draft"
	StatusInReview = "in_review"
	StatusActive   = "active"
	StatusArchived = "archived"

	APIVersion = "ash.harness/v1"
	KindSpec   = "HarnessProfileSpec"
	schemaID   = "https://ash.local/schemas/ash.harness.profile.v1.json"
)

type ProviderSpec struct {
	Kind  string `json:"kind"`
	Model string `json:"model,omitempty"`
}

type SandboxSpec struct {
	DefaultMode   string `json:"defaultMode"`
	Network       string `json:"network,omitempty"`
	SpillMaxBytes int    `json:"spillMaxBytes,omitempty"`
}

type ToolsSpec struct {
	Allowlist []string `json:"allowlist,omitempty"`
	Denylist  []string `json:"denylist,omitempty"`
}

type CompactionSpec struct {
	Enabled           bool    `json:"enabled,omitempty"`
	TriggerTokenRatio float64 `json:"triggerTokenRatio,omitempty"`
}

type SubRunSpec struct {
	MaxDepth       int  `json:"maxDepth,omitempty"`
	InheritSandbox bool `json:"inheritSandbox,omitempty"`
}

type IntegrationSpec struct {
	RPCEnabled        bool `json:"rpcEnabled,omitempty"`
	JSONEventsEnabled bool `json:"jsonEventsEnabled,omitempty"`
}

type ProfileSpecBody struct {
	Provider      ProviderSpec     `json:"provider"`
	Sandbox       SandboxSpec      `json:"sandbox"`
	Tools         ToolsSpec        `json:"tools"`
	PolicyProfile string           `json:"policyProfile"`
	Compaction    *CompactionSpec  `json:"compaction,omitempty"`
	SubRun        *SubRunSpec      `json:"subRun,omitempty"`
	Skills        []string         `json:"skills,omitempty"`
	Integration   *IntegrationSpec `json:"integration,omitempty"`
}

type ProfileDocument struct {
	APIVersion string          `json:"apiVersion"`
	Kind       string          `json:"kind"`
	Spec       ProfileSpecBody `json:"spec"`
}

type ProfileView struct {
	ID            string          `json:"id"`
	SpaceID       string          `json:"spaceId"`
	Name          string          `json:"name"`
	Version       int             `json:"version"`
	Status        string          `json:"status"`
	Spec          ProfileSpecBody `json:"spec"`
	ParentVersion *int            `json:"parentVersion,omitempty"`
	CreatedBy     string          `json:"createdBy,omitempty"`
	PromotedBy    string          `json:"promotedBy,omitempty"`
	CreatedAt     int64           `json:"createdAt"`
	UpdatedAt     int64           `json:"updatedAt"`
	PromotedAt    *int64          `json:"promotedAt,omitempty"`
}

type CreateRequest struct {
	SpaceID   string          `json:"spaceId"`
	Name      string          `json:"name"`
	Spec      ProfileSpecBody `json:"spec"`
	CreatedBy string          `json:"createdBy,omitempty"`
}

type UpdateRequest struct {
	Spec ProfileSpecBody `json:"spec"`
}

type Service struct {
	db *store.DB
}

func NewService(db *store.DB) *Service {
	return &Service{db: db}
}

func (s *Service) WithContext(ctx context.Context) *Service {
	if s == nil || ctx == nil {
		return s
	}
	return &Service{db: s.db.BindContext(ctx)}
}

func DefaultSpec() ProfileSpecBody {
	return ProfileSpecBody{
		Provider:      ProviderSpec{Kind: "execgo"},
		Sandbox:       SandboxSpec{DefaultMode: "workspace-write", Network: "deny", SpillMaxBytes: 65536},
		Tools:         ToolsSpec{Allowlist: []string{"read", "write", "edit", "bash", "grep", "find", "ls"}},
		PolicyProfile: "default",
		Compaction:    &CompactionSpec{Enabled: true, TriggerTokenRatio: 0.85},
		SubRun:        &SubRunSpec{MaxDepth: 2, InheritSandbox: true},
		Skills:        []string{},
		Integration:   &IntegrationSpec{RPCEnabled: true, JSONEventsEnabled: true},
	}
}

var (
	schemaOnce sync.Once
	schemaObj  *jsonschema.Schema
	schemaErr  error
)

func compiledSchema() (*jsonschema.Schema, error) {
	schemaOnce.Do(func() {
		doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(profileSchemaJSON))
		if err != nil {
			schemaErr = err
			return
		}
		c := jsonschema.NewCompiler()
		if err := c.AddResource(schemaID, doc); err != nil {
			schemaErr = err
			return
		}
		schemaObj, schemaErr = c.Compile(schemaID)
	})
	return schemaObj, schemaErr
}

func ValidateSpec(spec ProfileSpecBody) error {
	doc := ProfileDocument{APIVersion: APIVersion, Kind: KindSpec, Spec: spec}
	raw, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	sch, err := compiledSchema()
	if err != nil {
		return fmt.Errorf("compile schema: %w", err)
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return err
	}
	if err := sch.Validate(v); err != nil {
		return fmt.Errorf("spec invalid: %w", err)
	}
	return nil
}

func (s *Service) Create(req CreateRequest) (*ProfileView, error) {
	space := strings.TrimSpace(req.SpaceID)
	if space == "" {
		space = "local"
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	spec := req.Spec
	if spec.Provider.Kind == "" {
		spec = DefaultSpec()
	}
	if err := ValidateSpec(spec); err != nil {
		return nil, err
	}
	specJSON, err := json.Marshal(ProfileDocument{APIVersion: APIVersion, Kind: KindSpec, Spec: spec})
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	ver := 1
	var maxVer int
	_ = s.db.Model(&store.HarnessProfileVersion{}).
		Where("space_id = ? AND name = ?", space, name).
		Select("COALESCE(MAX(version), 0)").Scan(&maxVer)
	if maxVer > 0 {
		ver = maxVer + 1
	}
	row := store.HarnessProfileVersion{
		ID:        "hprof_" + uuid.NewString(),
		SpaceID:   space,
		Name:      name,
		Version:   ver,
		Status:    StatusDraft,
		SpecJSON:  string(specJSON),
		CreatedBy: strings.TrimSpace(req.CreatedBy),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.db.Create(&row).Error; err != nil {
		return nil, err
	}
	return toView(row)
}

func (s *Service) List(spaceID, status, name string) ([]ProfileView, error) {
	space := strings.TrimSpace(spaceID)
	if space == "" {
		space = "local"
	}
	q := s.db.Where("space_id = ?", space)
	if st := strings.TrimSpace(status); st != "" {
		q = q.Where("status = ?", st)
	}
	if n := strings.TrimSpace(name); n != "" {
		q = q.Where("name = ?", n)
	}
	var rows []store.HarnessProfileVersion
	if err := q.Order("name asc, version desc").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]ProfileView, 0, len(rows))
	for _, row := range rows {
		v, err := toView(row)
		if err != nil {
			return nil, err
		}
		out = append(out, *v)
	}
	return out, nil
}

func (s *Service) Get(id string) (*ProfileView, error) {
	var row store.HarnessProfileVersion
	if err := s.db.First(&row, "id = ?", strings.TrimSpace(id)).Error; err != nil {
		return nil, err
	}
	return toView(row)
}

func (s *Service) Update(id string, req UpdateRequest) (*ProfileView, error) {
	var row store.HarnessProfileVersion
	if err := s.db.First(&row, "id = ?", strings.TrimSpace(id)).Error; err != nil {
		return nil, err
	}
	if row.Status != StatusDraft {
		return nil, fmt.Errorf("only draft profiles can be updated")
	}
	if err := ValidateSpec(req.Spec); err != nil {
		return nil, err
	}
	specJSON, err := json.Marshal(ProfileDocument{APIVersion: APIVersion, Kind: KindSpec, Spec: req.Spec})
	if err != nil {
		return nil, err
	}
	row.SpecJSON = string(specJSON)
	row.UpdatedAt = time.Now().UTC()
	if err := s.db.Save(&row).Error; err != nil {
		return nil, err
	}
	return toView(row)
}

func (s *Service) SubmitReview(id string) (*ProfileView, error) {
	var row store.HarnessProfileVersion
	if err := s.db.First(&row, "id = ?", strings.TrimSpace(id)).Error; err != nil {
		return nil, err
	}
	if row.Status != StatusDraft {
		return nil, fmt.Errorf("only draft profiles can be submitted for review")
	}
	row.Status = StatusInReview
	row.UpdatedAt = time.Now().UTC()
	if err := s.db.Save(&row).Error; err != nil {
		return nil, err
	}
	return toView(row)
}

func (s *Service) Reject(id, actorID, reason string) (*ProfileView, error) {
	var row store.HarnessProfileVersion
	if err := s.db.First(&row, "id = ?", strings.TrimSpace(id)).Error; err != nil {
		return nil, err
	}
	if row.Status != StatusInReview && row.Status != StatusDraft {
		return nil, fmt.Errorf("profile must be draft or in_review to reject")
	}
	_ = actorID
	_ = reason
	row.Status = StatusArchived
	row.UpdatedAt = time.Now().UTC()
	if err := s.db.Save(&row).Error; err != nil {
		return nil, err
	}
	return toView(row)
}

func (s *Service) Promote(id, actorID string) (*ProfileView, error) {
	var row store.HarnessProfileVersion
	if err := s.db.First(&row, "id = ?", strings.TrimSpace(id)).Error; err != nil {
		return nil, err
	}
	if row.Status != StatusInReview {
		return nil, fmt.Errorf("profile must be in_review to promote (submit-review first)")
	}
	now := time.Now().UTC()
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&store.HarnessProfileVersion{}).
			Where("space_id = ? AND name = ? AND status = ? AND id <> ?", row.SpaceID, row.Name, StatusActive, row.ID).
			Updates(map[string]any{"status": StatusArchived, "updated_at": now}).Error; err != nil {
			return err
		}
		row.Status = StatusActive
		row.PromotedBy = strings.TrimSpace(actorID)
		row.PromotedAt = &now
		row.UpdatedAt = now
		return tx.Save(&row).Error
	})
	if err != nil {
		return nil, err
	}
	return toView(row)
}

// Rollback restores the latest archived version for the same (space,name) to active.
func (s *Service) Rollback(id, actorID string) (*ProfileView, error) {
	var current store.HarnessProfileVersion
	if err := s.db.First(&current, "id = ?", strings.TrimSpace(id)).Error; err != nil {
		return nil, err
	}
	if current.Status != StatusActive {
		return nil, fmt.Errorf("only active profiles can be rolled back")
	}
	var prev store.HarnessProfileVersion
	err := s.db.Where("space_id = ? AND name = ? AND status = ? AND id <> ?", current.SpaceID, current.Name, StatusArchived, current.ID).
		Order("version desc").First(&prev).Error
	if err != nil {
		return nil, fmt.Errorf("no archived version to rollback to: %w", err)
	}
	now := time.Now().UTC()
	err = s.db.Transaction(func(tx *gorm.DB) error {
		current.Status = StatusArchived
		current.UpdatedAt = now
		if err := tx.Save(&current).Error; err != nil {
			return err
		}
		prev.Status = StatusActive
		prev.PromotedBy = strings.TrimSpace(actorID)
		prev.PromotedAt = &now
		prev.UpdatedAt = now
		return tx.Save(&prev).Error
	})
	if err != nil {
		return nil, err
	}
	return toView(prev)
}

// LoadActive returns the active profile for space+name, or platform default when missing.
func (s *Service) LoadActive(spaceID, name string) (*ProfileView, error) {
	space := strings.TrimSpace(spaceID)
	if space == "" {
		space = "local"
	}
	n := strings.TrimSpace(name)
	if n == "" {
		n = "default"
	}
	var row store.HarnessProfileVersion
	err := s.db.Where("space_id = ? AND name = ? AND status = ?", space, n, StatusActive).
		Order("version desc").First(&row).Error
	if err == gorm.ErrRecordNotFound {
		spec := DefaultSpec()
		now := time.Now().UTC().UnixMilli()
		return &ProfileView{
			ID:        "hprof_platform_default",
			SpaceID:   space,
			Name:      n,
			Version:   0,
			Status:    StatusActive,
			Spec:      spec,
			CreatedAt: now,
			UpdatedAt: now,
		}, nil
	}
	if err != nil {
		return nil, err
	}
	return toView(row)
}

func toView(row store.HarnessProfileVersion) (*ProfileView, error) {
	var doc ProfileDocument
	if err := json.Unmarshal([]byte(row.SpecJSON), &doc); err != nil {
		return nil, fmt.Errorf("decode spec: %w", err)
	}
	v := &ProfileView{
		ID:            row.ID,
		SpaceID:       row.SpaceID,
		Name:          row.Name,
		Version:       row.Version,
		Status:        row.Status,
		Spec:          doc.Spec,
		ParentVersion: row.ParentVersion,
		CreatedBy:     row.CreatedBy,
		PromotedBy:    row.PromotedBy,
		CreatedAt:     row.CreatedAt.UTC().UnixMilli(),
		UpdatedAt:     row.UpdatedAt.UTC().UnixMilli(),
	}
	if row.PromotedAt != nil {
		ms := row.PromotedAt.UTC().UnixMilli()
		v.PromotedAt = &ms
	}
	return v, nil
}

// ActiveUniquenessOK reports whether each (space,name) has at most one active version.
func (s *Service) ActiveUniquenessOK(spaceID string) (bool, error) {
	type agg struct {
		Name  string
		Count int64
	}
	var rows []agg
	q := s.db.Model(&store.HarnessProfileVersion{}).
		Select("name, count(*) as count").
		Where("status = ?", StatusActive)
	if space := strings.TrimSpace(spaceID); space != "" {
		q = q.Where("space_id = ?", space)
	}
	if err := q.Group("space_id, name").Having("count(*) > 1").Scan(&rows).Error; err != nil {
		return false, err
	}
	return len(rows) == 0, nil
}
