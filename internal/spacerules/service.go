package spacerules

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
	"gorm.io/gorm"

	"github.com/ash-repwiki/ash/internal/store"
)

const RelativeFilePath = ".ash/rules.yaml"

// Document is the Space Rules payload (YAML on disk / DB body).
type Document struct {
	Version         int                 `json:"version" yaml:"version"`
	PreferScenario  string              `json:"preferScenario,omitempty" yaml:"preferScenario,omitempty"`
	Route           map[string][]string `json:"route,omitempty" yaml:"route,omitempty"`
	Defaults        Defaults            `json:"defaults,omitempty" yaml:"defaults,omitempty"`
}

type Defaults struct {
	PolicyProfile string         `json:"policyProfile,omitempty" yaml:"policyProfile,omitempty"`
	Inputs        map[string]any `json:"inputs,omitempty" yaml:"inputs,omitempty"`
}

type View struct {
	SpaceID   string    `json:"spaceId"`
	Version   int       `json:"version"`
	Source    string    `json:"source"`
	FilePath  string    `json:"filePath,omitempty"`
	UpdatedBy string    `json:"updatedBy,omitempty"`
	UpdatedAt int64     `json:"updatedAt"`
	Document  Document  `json:"document"`
	Builtin   bool      `json:"builtin"`
}

type PutRequest struct {
	Document  Document `json:"document"`
	UpdatedBy string   `json:"updatedBy,omitempty"`
}

type SyncRequest struct {
	RepoRoot  string `json:"repoRoot"`
	UpdatedBy string `json:"updatedBy,omitempty"`
}

type PreviewRequest struct {
	Goal     string `json:"goal"`
	RepoRoot string `json:"repoRoot"`
}

type PreviewResponse struct {
	ScenarioName  string         `json:"scenarioName"`
	RouteReason   string         `json:"routeReason"`
	PolicyProfile string         `json:"policyProfile"`
	Inputs        map[string]any `json:"inputs"`
}

type Service struct {
	db *store.DB
}

func NewService(db *store.DB) *Service {
	return &Service{db: db}
}

func (s *Service) gdb() *gorm.DB {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.DB
}

// BuiltinDocument returns DJ-compatible keyword defaults.
func BuiltinDocument() Document {
	return Document{
		Version: 1,
		Route: map[string][]string{
			"security_patch":    {"security", "cve", "vuln", "漏洞", "安全", "xss", "rce"},
			"hotfix":            {"hotfix", "urgent", "prod outage", "线上", "热修", "生产故障", "p0"},
			"feature_delivery":  {},
		},
		Defaults: Defaults{PolicyProfile: "default"},
	}
}

func (s *Service) Get(spaceID string) (*View, error) {
	spaceID = firstNonEmpty(spaceID, "local")
	var row store.SpaceRule
	err := s.gdb().Where("space_id = ?", spaceID).First(&row).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			doc := BuiltinDocument()
			return &View{
				SpaceID: spaceID, Version: doc.Version, Source: "default",
				Document: doc, Builtin: true, UpdatedAt: time.Now().UTC().Unix(),
			}, nil
		}
		return nil, err
	}
	doc, err := ParseDocument(row.BodyYAML)
	if err != nil {
		return nil, err
	}
	return rowToView(row, doc, false), nil
}

func (s *Service) Put(spaceID string, req PutRequest) (*View, error) {
	spaceID = firstNonEmpty(spaceID, "local")
	doc := NormalizeDocument(req.Document)
	body, err := yaml.Marshal(doc)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	var row store.SpaceRule
	err = s.gdb().Where("space_id = ?", spaceID).First(&row).Error
	if err == gorm.ErrRecordNotFound {
		row = store.SpaceRule{
			ID: "srule_" + uuid.NewString(), SpaceID: spaceID, Version: doc.Version,
			BodyYAML: string(body), Source: "api", UpdatedBy: strings.TrimSpace(req.UpdatedBy),
			CreatedAt: now, UpdatedAt: now,
		}
		if err := s.gdb().Create(&row).Error; err != nil {
			return nil, err
		}
		return rowToView(row, doc, false), nil
	}
	if err != nil {
		return nil, err
	}
	row.Version = doc.Version
	row.BodyYAML = string(body)
	row.Source = "api"
	row.UpdatedBy = strings.TrimSpace(req.UpdatedBy)
	row.UpdatedAt = now
	if err := s.gdb().Save(&row).Error; err != nil {
		return nil, err
	}
	return rowToView(row, doc, false), nil
}

func (s *Service) ImportFromFile(spaceID string, req SyncRequest) (*View, error) {
	spaceID = firstNonEmpty(spaceID, "local")
	path, err := resolveRulesPath(req.RepoRoot)
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	doc, err := ParseDocument(string(raw))
	if err != nil {
		return nil, err
	}
	doc = NormalizeDocument(doc)
	body, err := yaml.Marshal(doc)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	var row store.SpaceRule
	err = s.gdb().Where("space_id = ?", spaceID).First(&row).Error
	if err == gorm.ErrRecordNotFound {
		row = store.SpaceRule{
			ID: "srule_" + uuid.NewString(), SpaceID: spaceID, Version: doc.Version,
			BodyYAML: string(body), Source: "file", FilePath: RelativeFilePath,
			UpdatedBy: strings.TrimSpace(req.UpdatedBy), CreatedAt: now, UpdatedAt: now,
		}
		if err := s.gdb().Create(&row).Error; err != nil {
			return nil, err
		}
		return rowToView(row, doc, false), nil
	}
	if err != nil {
		return nil, err
	}
	row.Version = doc.Version
	row.BodyYAML = string(body)
	row.Source = "file"
	row.FilePath = RelativeFilePath
	row.UpdatedBy = strings.TrimSpace(req.UpdatedBy)
	row.UpdatedAt = now
	if err := s.gdb().Save(&row).Error; err != nil {
		return nil, err
	}
	return rowToView(row, doc, false), nil
}

func (s *Service) ExportToFile(spaceID string, req SyncRequest) (*View, error) {
	view, err := s.Get(spaceID)
	if err != nil {
		return nil, err
	}
	if view.Builtin {
		return nil, fmt.Errorf("no persisted rules to export; save rules first")
	}
	path, err := resolveRulesPath(req.RepoRoot)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	body, err := yaml.Marshal(view.Document)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, body, 0o644); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	_ = s.gdb().Model(&store.SpaceRule{}).Where("space_id = ?", firstNonEmpty(spaceID, "local")).
		Updates(map[string]any{"file_path": RelativeFilePath, "source": "api", "updated_at": now}).Error
	view.FilePath = RelativeFilePath
	view.UpdatedAt = now.Unix()
	return view, nil
}

func ParseDocument(raw string) (Document, error) {
	var doc Document
	if strings.TrimSpace(raw) == "" {
		return BuiltinDocument(), nil
	}
	if err := yaml.Unmarshal([]byte(raw), &doc); err != nil {
		return Document{}, fmt.Errorf("invalid rules yaml: %w", err)
	}
	return NormalizeDocument(doc), nil
}

func NormalizeDocument(doc Document) Document {
	if doc.Version <= 0 {
		doc.Version = 1
	}
	if doc.Route == nil {
		doc.Route = map[string][]string{}
	}
	builtin := BuiltinDocument()
	for name, keys := range builtin.Route {
		if _, ok := doc.Route[name]; !ok {
			doc.Route[name] = append([]string(nil), keys...)
		}
	}
	if doc.Defaults.PolicyProfile == "" {
		doc.Defaults.PolicyProfile = "default"
	}
	if doc.Defaults.Inputs == nil {
		doc.Defaults.Inputs = map[string]any{}
	}
	return doc
}

func resolveRulesPath(repoRoot string) (string, error) {
	root := strings.TrimSpace(repoRoot)
	if root == "" {
		root = "."
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	return filepath.Join(abs, RelativeFilePath), nil
}

func rowToView(row store.SpaceRule, doc Document, builtin bool) *View {
	return &View{
		SpaceID: row.SpaceID, Version: row.Version, Source: row.Source,
		FilePath: row.FilePath, UpdatedBy: row.UpdatedBy,
		UpdatedAt: row.UpdatedAt.Unix(), Document: doc, Builtin: builtin,
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
