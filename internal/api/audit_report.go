package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ash-repwiki/ash/internal/store"
)

// AuditReportBuckets are the DX69 thin slices over existing audit_log rows.
type AuditReportBuckets struct {
	Approve int64 `json:"approve"`
	Deny    int64 `json:"deny"`
	Hook    int64 `json:"hook"`
	Spawn   int64 `json:"spawn"`
	Other   int64 `json:"other"`
}

// AuditEventCount is one event_type total inside the window.
type AuditEventCount struct {
	EventType string `json:"eventType"`
	Count     int64  `json:"count"`
}

// AuditReport is GET /spaces/{id}/audit-report.
type AuditReport struct {
	SpaceID  string             `json:"spaceId"`
	Window   string             `json:"window"`
	From     time.Time          `json:"from"`
	Total    int64              `json:"total"`
	Buckets  AuditReportBuckets `json:"buckets"`
	ByEvent  []AuditEventCount  `json:"byEvent"`
}

func auditWindow(raw string) (string, time.Duration, error) {
	switch strings.TrimSpace(raw) {
	case "", "7d":
		return "7d", 7 * 24 * time.Hour, nil
	case "24h":
		return "24h", 24 * time.Hour, nil
	case "30d":
		return "30d", 30 * 24 * time.Hour, nil
	default:
		return "", 0, errInvalidAuditWindow
	}
}

var errInvalidAuditWindow = errString("window must be 24h, 7d, or 30d")

type errString string

func (e errString) Error() string { return string(e) }

func auditBucket(eventType string) string {
	switch {
	case strings.HasPrefix(eventType, "hook."):
		return "hook"
	case strings.Contains(eventType, "spawn") || strings.Contains(eventType, "subrun") || strings.Contains(eventType, "sub-run"):
		return "spawn"
	case strings.HasSuffix(eventType, ".denied") || strings.HasSuffix(eventType, ".rejected"):
		return "deny"
	case strings.HasSuffix(eventType, ".approved") || eventType == "gate.approved":
		return "approve"
	default:
		return "other"
	}
}

// getSpaceAuditReport godoc
// @Summary Thin space audit report
// @Description DX69: counts existing audit_log rows in a window. No new table.
// @Tags spaces
// @Produce json
// @Param spaceId path string true "space id"
// @Param window query string false "24h, 7d (default), or 30d"
// @Success 200 {object} AuditReport
// @Failure 400 {object} APIErrorResponse
// @Failure 403 {object} APIErrorResponse
// @Router /api/v1/spaces/{spaceId}/audit-report [get]
func (h *Handler) getSpaceAuditReport(c *gin.Context) {
	spaceID := strings.TrimSpace(c.Param("spaceId"))
	if spaceID == "" {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", "spaceId is required"))
		return
	}
	window, dur, err := auditWindow(c.Query("window"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_WINDOW", err.Error()))
		return
	}
	if !h.requirePermission(c, permAuditExport, spaceID) {
		return
	}
	from := time.Now().UTC().Add(-dur)
	var rows []struct {
		EventType string
		Count     int64
	}
	if err := h.dbFor(c).Model(&store.AuditLog{}).
		Select("event_type, COUNT(*) as count").
		Where("space_id = ? AND created_at >= ?", spaceID, from).
		Group("event_type").
		Scan(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("AUDIT_REPORT_FAILED", err.Error()))
		return
	}
	report := AuditReport{
		SpaceID: spaceID,
		Window:  window,
		From:    from,
		ByEvent: []AuditEventCount{},
	}
	for _, row := range rows {
		report.Total += row.Count
		report.ByEvent = append(report.ByEvent, AuditEventCount{EventType: row.EventType, Count: row.Count})
		switch auditBucket(row.EventType) {
		case "approve":
			report.Buckets.Approve += row.Count
		case "deny":
			report.Buckets.Deny += row.Count
		case "hook":
			report.Buckets.Hook += row.Count
		case "spawn":
			report.Buckets.Spawn += row.Count
		default:
			report.Buckets.Other += row.Count
		}
	}
	c.JSON(http.StatusOK, report)
}
