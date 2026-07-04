package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ash-repwiki/ash/internal/alerts"
	"github.com/ash-repwiki/ash/internal/ci"
	"github.com/ash-repwiki/ash/internal/events"
	"github.com/ash-repwiki/ash/internal/improve"
	metricssvc "github.com/ash-repwiki/ash/internal/metrics"
	"github.com/ash-repwiki/ash/internal/releases"
	"github.com/ash-repwiki/ash/internal/runs"
	"github.com/ash-repwiki/ash/internal/store"
	"gorm.io/gorm"
)

// rlsMiddleware binds tenant space or org-admin bypass to the request context for Postgres RLS.
func (h *Handler) rlsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !store.PostgresRLSEnabled() || h.db == nil || h.db.Dialect() != "postgres" {
			c.Next()
			return
		}
		ctx := c.Request.Context()
		bypass := h.rlsBypassForRequest(c)
		if bypass {
			ctx = store.WithRLSBypassContext(ctx)
		} else {
			spaceID := currentSpace(c)
			ctx = store.WithRLSSpaceContext(ctx, spaceID)
			if orgID, err := h.lookupSpaceOrgID(c, spaceID); err == nil && orgID != "" {
				ctx = store.WithRLSOrgContext(ctx, orgID)
			}
		}
		c.Request = c.Request.WithContext(ctx)
		c.Next()
		if bypass {
			_ = h.dbFor(c).Create(auditRow(currentSpace(c), currentActor(c), "tenant.rls_bypass", map[string]any{
				"path": c.FullPath(), "method": c.Request.Method,
			})).Error
		}
	}
}

func rlsBypassRoutePath(c *gin.Context) string {
	if p := c.FullPath(); p != "" {
		return p
	}
	if c.Request != nil && c.Request.URL != nil {
		return c.Request.URL.Path
	}
	return ""
}

// rlsBypassRouteMatch reports whether the route may use org-admin RLS bypass (before permission check).
func (h *Handler) rlsBypassRouteMatch(c *gin.Context) bool {
	if c.Request.Method != http.MethodGet {
		return false
	}
	switch rlsBypassRoutePath(c) {
	case "/api/v1/orgs", "/api/v1/spaces":
		return true
	default:
		return false
	}
}

// rlsBypassForRequest allows org admins to list orgs/spaces across tenant RLS (audited).
func (h *Handler) rlsBypassForRequest(c *gin.Context) bool {
	if !h.rlsBypassRouteMatch(c) {
		return false
	}
	ok, err := h.hasPermission(c, currentSpace(c), permOrgWrite)
	return err == nil && ok
}

// dbFor returns a GORM handle with the request context (required for Postgres RLS session vars).
func (h *Handler) dbFor(c *gin.Context) *gorm.DB {
	return h.db.WithContext(c.Request.Context())
}

// dbBypass returns a GORM handle with migration/admin RLS bypass for the request.
func (h *Handler) dbBypass(c *gin.Context) *gorm.DB {
	return h.db.WithContext(store.WithRLSBypassContext(c.Request.Context()))
}

func (h *Handler) runsFor(c *gin.Context) *runs.Service {
	if h == nil || h.runs == nil {
		return h.runs
	}
	return h.runs.WithContext(c.Request.Context())
}

func (h *Handler) eventsFor(c *gin.Context) *events.Service {
	if h == nil || h.events == nil {
		return h.events
	}
	return h.events.WithContext(c.Request.Context())
}

func (h *Handler) ciFor(c *gin.Context) *ci.Service {
	if h == nil || h.ci == nil {
		return h.ci
	}
	return h.ci.WithContext(c.Request.Context())
}

func (h *Handler) alertsFor(c *gin.Context) *alerts.Service {
	if h == nil || h.alerts == nil {
		return h.alerts
	}
	return h.alerts.WithContext(c.Request.Context())
}

func (h *Handler) releasesFor(c *gin.Context) *releases.Service {
	if h == nil || h.releases == nil {
		return h.releases
	}
	return h.releases.WithContext(c.Request.Context())
}

func (h *Handler) improveFor(c *gin.Context) *improve.Service {
	if h == nil || h.improve == nil {
		return h.improve
	}
	return h.improve.WithContext(c.Request.Context())
}

func (h *Handler) metricsFor(c *gin.Context) *metricssvc.Service {
	if h == nil || h.metrics == nil {
		return h.metrics
	}
	return h.metrics.WithContext(c.Request.Context())
}

func (h *Handler) lookupSpaceOrgID(c *gin.Context, spaceID string) (string, error) {
	spaceID = strings.TrimSpace(spaceID)
	if spaceID == "" || h == nil || h.db == nil {
		return "", nil
	}
	var row struct {
		OrgID string
	}
	err := h.dbBypass(c).Table("spaces").Select("org_id").Where("id = ?", spaceID).Scan(&row).Error
	return strings.TrimSpace(row.OrgID), err
}
