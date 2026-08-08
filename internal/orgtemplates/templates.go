// Package orgtemplates defines commercial organization landing templates (PRD §3).
package orgtemplates

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ash-repwiki/ash/internal/authz"
	"github.com/ash-repwiki/ash/internal/store"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Template IDs are stable API keys.
const (
	IDSmallTeam        = "small_team"
	IDMidEnterprise    = "mid_enterprise"
	IDStrongCompliance = "strong_compliance"
)

// RoleSpec is a role to create under the org (beyond the always-present admin).
type RoleSpec struct {
	Name        string   `json:"name"`
	Permissions []string `json:"permissions"`
}

// SpaceSpec is a space created by the template.
type SpaceSpec struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// Template is a catalog entry for org commercial landing.
type Template struct {
	ID              string      `json:"id"`
	Label           string      `json:"label"`
	Description     string      `json:"description"`
	Deployment      string      `json:"deployment"` // internal_platform | saas | either
	Payer           string      `json:"payer"`
	DecisionMaker   string      `json:"decisionMaker"`
	Approver        string      `json:"approver"`
	DefaultOrgName  string      `json:"defaultOrgName"`
	DefaultOrgSlug  string      `json:"defaultOrgSlug"`
	Spaces          []SpaceSpec `json:"spaces"`
	ExtraRoles      []RoleSpec  `json:"extraRoles"`
	RecommendedKPIs []string    `json:"recommendedKpis"`
	Scenarios       []string    `json:"scenarios"`
}

// Catalog returns the three PRD §3 organization templates.
func Catalog() []Template {
	return []Template{
		{
			ID:             IDSmallTeam,
			Label:          "小团队 / 创业交付组",
			Description:    "单 Space 全速交付；组织管理员兼任决策与审批；适合内部平台或轻量 SaaS。",
			Deployment:     "either",
			Payer:          "工程负责人（团队预算）",
			DecisionMaker:  "Tech Lead / 工程负责人",
			Approver:       "同一 Tech Lead（门禁 human step）",
			DefaultOrgName: "Startup Delivery",
			DefaultOrgSlug: "startup-delivery",
			Spaces:         []SpaceSpec{{Name: "Delivery", Slug: "delivery"}},
			ExtraRoles: []RoleSpec{
				{Name: "operator", Permissions: []string{
					"run:create", "run:cancel", "artifact:read", "memory:create", "memory:read", "memory:query",
					"rag:query", "ci:read", "ci:diagnose", "feedback:read", "feedback:write", "observability:read",
				}},
				{Name: "reviewer", Permissions: []string{
					"run:approve", "artifact:read", "memory:read", "memory:review", "memory:query",
					"ci:read", "feedback:read", "feedback:write", "observability:read",
				}},
			},
			RecommendedKPIs: []string{"KPI-01", "KPI-02", "KPI-06", "KPI-11"},
			Scenarios:       []string{"feature_delivery@1.0.0", "hotfix@1.1.0"},
		},
		{
			ID:             IDMidEnterprise,
			Label:          "中大型研发组织",
			Description:    "平台组付费、事业部决策、发布窗审批分离；多 Space（交付 / 平台）隔离。",
			Deployment:     "internal_platform",
			Payer:          "工程效率 / 平台预算 Owner",
			DecisionMaker:  "事业部研发负责人 + 架构委员会",
			Approver:       "Release Manager（发版）+ Reviewer（合并）",
			DefaultOrgName: "Enterprise Engineering",
			DefaultOrgSlug: "enterprise-eng",
			Spaces: []SpaceSpec{
				{Name: "Product Delivery", Slug: "product-delivery"},
				{Name: "Platform", Slug: "platform"},
			},
			ExtraRoles: []RoleSpec{
				{Name: "operator", Permissions: []string{
					"run:create", "run:cancel", "artifact:read", "memory:create", "memory:read", "memory:query",
					"rag:query", "repo:read", "ci:read", "ci:diagnose", "feedback:read", "feedback:write",
					"observability:read", "release:read",
				}},
				{Name: "reviewer", Permissions: []string{
					"run:approve", "artifact:read", "memory:read", "memory:review", "memory:query",
					"ci:read", "feedback:read", "release:read", "observability:read",
				}},
				{Name: "release_manager", Permissions: []string{
					"run:approve", "artifact:read", "release:read", "release:write", "ci:read",
					"audit:export", "observability:read", "feedback:read",
				}},
				{Name: "librarian", Permissions: []string{
					"memory:create", "memory:read", "memory:review", "memory:query", "memory:use",
					"rag:index", "rag:query", "feedback:read", "artifact:read",
				}},
			},
			RecommendedKPIs: []string{"KPI-01", "KPI-03", "KPI-04", "KPI-05", "KPI-07", "KPI-11"},
			Scenarios:       []string{"feature_delivery@1.0.0", "hotfix@1.1.0", "security_patch@1.1.0"},
		},
		{
			ID:             IDStrongCompliance,
			Label:          "强合规 / 受监管交付",
			Description:    "合规预算与安全审批主导；审计与密钥权限收紧；适合金融/政务内场或私有化。",
			Deployment:     "internal_platform",
			Payer:          "合规 / 信息安全委员会预算",
			DecisionMaker:  "CISO / 合规官 + 业务 Owner",
			Approver:       "Security approver + 双人发版门禁",
			DefaultOrgName: "Regulated Delivery",
			DefaultOrgSlug: "regulated-delivery",
			Spaces: []SpaceSpec{
				{Name: "Controlled Delivery", Slug: "controlled"},
				{Name: "Audit", Slug: "audit"},
			},
			ExtraRoles: []RoleSpec{
				{Name: "operator", Permissions: []string{
					"run:create", "run:cancel", "artifact:read", "memory:read", "memory:query",
					"ci:read", "feedback:read", "observability:read", "release:read",
				}},
				{Name: "reviewer", Permissions: []string{
					"run:approve", "artifact:read", "memory:read", "memory:review", "ci:read",
					"feedback:read", "audit:export", "observability:read", "release:read",
				}},
				{Name: "security", Permissions: []string{
					"run:approve", "artifact:read", "secret:read", "secret:write", "audit:export",
					"memory:read", "ci:read", "release:read", "release:write", "observability:read",
					"plugin:read",
				}},
				{Name: "auditor", Permissions: []string{
					"artifact:read", "audit:export", "feedback:read", "observability:read",
					"release:read", "ci:read", "memory:read", "member:read", "role:read",
				}},
			},
			RecommendedKPIs: []string{"KPI-01", "KPI-06", "KPI-08", "KPI-09", "KPI-11"},
			Scenarios:       []string{"security_patch@1.1.0", "hotfix@1.1.0", "feature_delivery@1.0.0"},
		},
	}
}

// Get returns a template by id.
func Get(id string) (Template, bool) {
	id = strings.TrimSpace(id)
	for _, t := range Catalog() {
		if t.ID == id {
			return t, true
		}
	}
	return Template{}, false
}

// ProvisionRequest customizes names when applying a template.
type ProvisionRequest struct {
	OrgName string
	OrgSlug string
	ActorID string
	SpaceID string // audit space context (often "local")
}

// ProvisionResult is the created org graph.
type ProvisionResult struct {
	TemplateID string        `json:"templateId"`
	Org        store.Org     `json:"org"`
	Roles      []store.Role  `json:"roles"`
	Spaces     []store.Space `json:"spaces"`
}

// Provision creates org + admin + extra roles + spaces with scenario scopes.
func Provision(tx *gorm.DB, templateID string, req ProvisionRequest) (ProvisionResult, error) {
	tpl, ok := Get(templateID)
	if !ok {
		return ProvisionResult{}, fmt.Errorf("unknown org template: %s", templateID)
	}
	now := time.Now().UTC()
	actor := strings.TrimSpace(req.ActorID)
	if actor == "" {
		actor = "dev-user"
	}
	orgName := firstNonEmpty(strings.TrimSpace(req.OrgName), tpl.DefaultOrgName)
	orgSlug := firstNonEmpty(strings.TrimSpace(req.OrgSlug), tpl.DefaultOrgSlug)
	orgSlug = slugify(orgSlug)

	org := store.Org{
		ID: "org_" + uuid.NewString(), Name: orgName, Slug: orgSlug,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := tx.Create(&org).Error; err != nil {
		return ProvisionResult{}, err
	}
	user := store.User{ID: actor, DisplayName: actor, Status: "active", CreatedAt: now, UpdatedAt: now}
	if err := tx.FirstOrCreate(&user, "id = ?", actor).Error; err != nil {
		return ProvisionResult{}, err
	}
	admin := store.Role{
		ID: "role_" + uuid.NewString(), OrgID: org.ID, Name: "admin",
		Permissions: `["*"]`, CreatedAt: now, UpdatedAt: now,
	}
	if err := tx.Create(&admin).Error; err != nil {
		return ProvisionResult{}, err
	}
	roles := []store.Role{admin}
	for _, spec := range tpl.ExtraRoles {
		raw, _ := json.Marshal(spec.Permissions)
		role := store.Role{
			ID: "role_" + uuid.NewString(), OrgID: org.ID, Name: spec.Name,
			Permissions: string(raw), CreatedAt: now, UpdatedAt: now,
		}
		if err := tx.Create(&role).Error; err != nil {
			return ProvisionResult{}, err
		}
		roles = append(roles, role)
	}
	member := store.Member{
		ID: "mem_" + uuid.NewString(), OrgID: org.ID, UserID: actor, RoleID: admin.ID,
		Status: "active", CreatedAt: now, UpdatedAt: now,
	}
	if err := tx.Create(&member).Error; err != nil {
		return ProvisionResult{}, err
	}

	spaces := make([]store.Space, 0, len(tpl.Spaces))
	for _, spec := range tpl.Spaces {
		space := store.Space{
			ID: "space_" + uuid.NewString(), OrgID: org.ID,
			Name: spec.Name, Slug: firstNonEmpty(spec.Slug, slugify(spec.Name)),
			CreatedAt: now, UpdatedAt: now,
		}
		if err := tx.Create(&space).Error; err != nil {
			return ProvisionResult{}, err
		}
		if err := tx.Create(&store.ResourceScope{
			ID: "scope_" + uuid.NewString(), SpaceID: space.ID,
			ResourceType: "space", ResourceID: space.ID, PolicyJSON: `{"inheritsOrg":true}`,
			CreatedAt: now, UpdatedAt: now,
		}).Error; err != nil {
			return ProvisionResult{}, err
		}
		if err := authz.SeedScenarioScopesTx(tx, space.ID, now); err != nil {
			return ProvisionResult{}, err
		}
		spaces = append(spaces, space)
	}

	auditSpace := firstNonEmpty(strings.TrimSpace(req.SpaceID), "local")
	if len(spaces) > 0 {
		auditSpace = spaces[0].ID
	}
	payload, _ := json.Marshal(map[string]any{
		"templateId": tpl.ID, "orgId": org.ID, "slug": org.Slug,
		"spaceIds": spaceIDs(spaces), "roleCount": len(roles),
	})
	if err := tx.Create(&store.AuditLog{
		ID: "aud_" + uuid.NewString(), SpaceID: auditSpace, ActorID: actor,
		EventType: "org.template_provisioned", PayloadJSON: string(payload), CreatedAt: now,
	}).Error; err != nil {
		return ProvisionResult{}, err
	}

	return ProvisionResult{TemplateID: tpl.ID, Org: org, Roles: roles, Spaces: spaces}, nil
}

func spaceIDs(spaces []store.Space) []string {
	out := make([]string, 0, len(spaces))
	for _, s := range spaces {
		out = append(out, s.ID)
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	prevDash := false
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			prevDash = false
		case r == ' ' || r == '_' || r == '-':
			if !prevDash && b.Len() > 0 {
				b.WriteByte('-')
				prevDash = true
			}
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "org"
	}
	return out
}
