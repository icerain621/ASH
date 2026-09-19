package spacepolicy

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/ash-repwiki/ash/internal/execpolicy"
	"github.com/ash-repwiki/ash/internal/hooks"
	"github.com/ash-repwiki/ash/internal/store"
)

const (
	CitationOptional = "optional"
	CitationRequired = "required"
	CitationStrict   = "strict"
)

// Pack is the persisted SpacePolicyPack body.
type Pack struct {
	SpaceID        string `json:"spaceId"`
	CitationMode   string `json:"citationMode"`
	MultiSign      bool   `json:"multiSign"`
	ReviewSLAHours int    `json:"reviewSlaHours"`
	BodyJSON       string `json:"bodyJson,omitempty"`
	UpdatedAt      int64  `json:"updatedAt,omitempty"`
}

// EffectivePolicy is the merged result: kind defaults < pack < ResourceScope.
type EffectivePolicy struct {
	SpaceID        string `json:"spaceId"`
	SpaceKind      string `json:"spaceKind"`
	CitationMode   string `json:"citationMode"`
	MultiSign      bool   `json:"multiSign"`
	ReviewSLAHours int    `json:"reviewSlaHours"`
	Sources        []string `json:"sources"`
}

type PutPackRequest struct {
	CitationMode   string `json:"citationMode"`
	MultiSign      *bool  `json:"multiSign"`
	ReviewSLAHours *int   `json:"reviewSlaHours"`
	BodyJSON       string `json:"bodyJson,omitempty"`
}

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

func (s *Service) GetPack(spaceID string) (*Pack, error) {
	space := firstNonEmpty(strings.TrimSpace(spaceID), "local")
	var row store.SpacePolicyPack
	err := s.q().First(&row, "space_id = ?", space).Error
	if err == gorm.ErrRecordNotFound {
		return &Pack{
			SpaceID: space, CitationMode: CitationOptional,
			MultiSign: false, ReviewSLAHours: 72, BodyJSON: "{}",
		}, nil
	}
	if err != nil {
		return nil, err
	}
	return packFromRow(row), nil
}

func (s *Service) PutPack(spaceID string, req PutPackRequest) (*Pack, error) {
	space := firstNonEmpty(strings.TrimSpace(spaceID), "local")
	now := time.Now().UTC()
	var row store.SpacePolicyPack
	err := s.q().First(&row, "space_id = ?", space).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	if err == gorm.ErrRecordNotFound {
		row = store.SpacePolicyPack{
			SpaceID: space, CitationMode: CitationOptional,
			MultiSign: false, ReviewSLAHours: 72, BodyJSON: "{}",
			CreatedAt: now, UpdatedAt: now,
		}
	}
	if mode := strings.ToLower(strings.TrimSpace(req.CitationMode)); mode != "" {
		if !validCitation(mode) {
			return nil, fmt.Errorf("citationMode must be optional|required|strict")
		}
		row.CitationMode = mode
	}
	if req.MultiSign != nil {
		row.MultiSign = *req.MultiSign
	}
	if req.ReviewSLAHours != nil {
		if *req.ReviewSLAHours < 1 {
			return nil, fmt.Errorf("reviewSlaHours must be >= 1")
		}
		row.ReviewSLAHours = *req.ReviewSLAHours
	}
	if req.BodyJSON != "" {
		if err := validateBodyJSON(req.BodyJSON); err != nil {
			return nil, err
		}
		row.BodyJSON = req.BodyJSON
	}
	if row.BodyJSON == "" {
		row.BodyJSON = "{}"
	}
	row.UpdatedAt = now
	if err == gorm.ErrRecordNotFound {
		if err := s.q().Create(&row).Error; err != nil {
			return nil, err
		}
	} else {
		if err := s.q().Save(&row).Error; err != nil {
			return nil, err
		}
	}
	return packFromRow(row), nil
}

// EffectivePolicy merges kind defaults < pack < ResourceScope.PolicyJSON (stricter wins).
func (s *Service) EffectivePolicy(spaceID string) (*EffectivePolicy, error) {
	space := firstNonEmpty(strings.TrimSpace(spaceID), "local")
	kind := "team"
	var sp store.Space
	if err := s.q().First(&sp, "id = ?", space).Error; err == nil && strings.TrimSpace(sp.Kind) != "" {
		kind = strings.ToLower(strings.TrimSpace(sp.Kind))
	}

	base := KindDefaults(kind)
	sources := []string{"kind:" + kind}

	pack, err := s.GetPack(space)
	if err != nil {
		return nil, err
	}
	merged := MergeStricter(base, PolicyFragment{
		CitationMode:   pack.CitationMode,
		MultiSign:      pack.MultiSign,
		ReviewSLAHours: pack.ReviewSLAHours,
	})
	if pack.UpdatedAt > 0 || pack.CitationMode != "" {
		sources = append(sources, "pack")
	}

	var scopes []store.ResourceScope
	_ = s.q().Where("space_id = ?", space).Find(&scopes).Error
	for _, sc := range scopes {
		frag := ParseScopePolicy(sc.PolicyJSON)
		merged = MergeStricter(merged, frag)
		if sc.PolicyJSON != "" && sc.PolicyJSON != "{}" {
			sources = append(sources, "scope:"+sc.ResourceType)
		}
	}

	return &EffectivePolicy{
		SpaceID: space, SpaceKind: kind,
		CitationMode: merged.CitationMode, MultiSign: merged.MultiSign,
		ReviewSLAHours: merged.ReviewSLAHours, Sources: sources,
	}, nil
}

// PolicyFragment is a partial policy used for merge.
type PolicyFragment struct {
	CitationMode   string
	MultiSign      bool
	ReviewSLAHours int
}

func KindDefaults(kind string) PolicyFragment {
	kind = strings.ToLower(strings.TrimSpace(kind))
	if kind == "user" {
		return PolicyFragment{
			CitationMode: CitationOptional, MultiSign: false, ReviewSLAHours: 168,
		}
	}
	// team: stricter defaults
	return PolicyFragment{
		CitationMode: CitationRequired, MultiSign: true, ReviewSLAHours: 72,
	}
}

// MergeStricter prefers required>optional citation, multiSign=true, lower SLA hours.
func MergeStricter(base, overlay PolicyFragment) PolicyFragment {
	out := base
	if citationRank(overlay.CitationMode) > citationRank(out.CitationMode) {
		out.CitationMode = overlay.CitationMode
	}
	if overlay.MultiSign {
		out.MultiSign = true
	}
	if overlay.ReviewSLAHours > 0 && (out.ReviewSLAHours == 0 || overlay.ReviewSLAHours < out.ReviewSLAHours) {
		out.ReviewSLAHours = overlay.ReviewSLAHours
	}
	return out
}

func ParseScopePolicy(raw string) PolicyFragment {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "{}" {
		return PolicyFragment{}
	}
	var m struct {
		CitationMode   string `json:"citationMode"`
		MultiSign      *bool  `json:"multiSign"`
		ReviewSLAHours *int   `json:"reviewSlaHours"`
	}
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return PolicyFragment{}
	}
	frag := PolicyFragment{CitationMode: strings.ToLower(strings.TrimSpace(m.CitationMode))}
	if m.MultiSign != nil {
		frag.MultiSign = *m.MultiSign
	}
	if m.ReviewSLAHours != nil {
		frag.ReviewSLAHours = *m.ReviewSLAHours
	}
	return frag
}

func citationRank(mode string) int {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case CitationStrict:
		return 3
	case CitationRequired:
		return 2
	case CitationOptional:
		return 1
	default:
		return 0
	}
}

func validCitation(mode string) bool {
	switch mode {
	case CitationOptional, CitationRequired, CitationStrict:
		return true
	default:
		return false
	}
}

func packFromRow(row store.SpacePolicyPack) *Pack {
	return &Pack{
		SpaceID: row.SpaceID, CitationMode: row.CitationMode,
		MultiSign: row.MultiSign, ReviewSLAHours: row.ReviewSLAHours,
		BodyJSON: row.BodyJSON, UpdatedAt: row.UpdatedAt.UTC().UnixMilli(),
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

// ToolApprovalPreset is a BodyJSON toolApprovalPresets entry (fail-closed: default ask).
type ToolApprovalPreset struct {
	Tool    string `json:"tool"`
	Risk    string `json:"risk,omitempty"`
	Default string `json:"default"`
}

const (
	PresetDefaultAsk   = "ask"
	PresetDefaultAllow = "allow"
	PresetDefaultDeny  = "deny"
)

// HooksConfigFromBodyJSON reads ash.hooks.v1 from SpacePolicy BodyJSON (missing hooks → empty config).
func HooksConfigFromBodyJSON(bodyJSON string) (hooks.Config, error) {
	return hooks.ConfigFromSpaceBodyJSON(bodyJSON)
}

// ExecPolicyFromBodyJSON reads ash.execpolicy.v1 from SpacePolicy BodyJSON (missing execPolicy → empty).
func ExecPolicyFromBodyJSON(bodyJSON string) (execpolicy.Policy, error) {
	return execpolicy.FromSpaceBodyJSON(bodyJSON)
}

// ParseToolApprovalPresets extracts toolApprovalPresets from SpacePolicy BodyJSON.
// Invalid JSON or missing field returns nil (fail-closed: no auto-allow).
func ParseToolApprovalPresets(bodyJSON string) []ToolApprovalPreset {
	bodyJSON = strings.TrimSpace(bodyJSON)
	if bodyJSON == "" || bodyJSON == "{}" {
		return nil
	}
	var body struct {
		ToolApprovalPresets []ToolApprovalPreset `json:"toolApprovalPresets"`
	}
	if err := json.Unmarshal([]byte(bodyJSON), &body); err != nil {
		return nil
	}
	out := make([]ToolApprovalPreset, 0, len(body.ToolApprovalPresets))
	for _, p := range body.ToolApprovalPresets {
		tool := strings.TrimSpace(p.Tool)
		if tool == "" {
			continue
		}
		def := strings.ToLower(strings.TrimSpace(p.Default))
		if def == "" {
			def = PresetDefaultAsk
		}
		out = append(out, ToolApprovalPreset{
			Tool: tool, Risk: strings.TrimSpace(p.Risk), Default: def,
		})
	}
	return out
}

// PresetAllowsTool reports whether BodyJSON explicitly allows tool without a gate.
// Only default=allow auto-skips; ask/deny/unknown stay fail-closed.
func PresetAllowsTool(bodyJSON, tool string) bool {
	tool = strings.TrimSpace(tool)
	if tool == "" {
		return false
	}
	for _, p := range ParseToolApprovalPresets(bodyJSON) {
		if p.Tool == tool && p.Default == PresetDefaultAllow {
			return true
		}
	}
	return false
}

// PutPack validates BodyJSON when present (must be object JSON; presets optional).
func validateBodyJSON(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return fmt.Errorf("bodyJson must be a JSON object: %w", err)
	}
	// Touch presets parse path so invalid nested types do not panic later.
	_ = ParseToolApprovalPresets(raw)
	// Fail-closed: if execPolicy is present it must parse as ash.execpolicy.v1.
	if _, err := ExecPolicyFromBodyJSON(raw); err != nil {
		return fmt.Errorf("bodyJson.execPolicy: %w", err)
	}
	return nil
}
