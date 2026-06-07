package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ash-repwiki/ash/internal/alerts"
	"github.com/ash-repwiki/ash/internal/store"
)

type AlertEventListResponse struct {
	Items []store.AlertEvent `json:"items"`
}

type AlertRuleListResponse struct {
	Items []store.AlertRule `json:"items"`
}

type putAlertRulesRequest struct {
	Items []alerts.RuleInput `json:"items"`
}

// PrometheusMetrics godoc
// @Summary Get deployment-global Prometheus metrics (ops scrape)
// @Description Unauthenticated scrape endpoint. When Postgres RLS is enabled, queries use admin bypass so Prometheus sees all spaces. For tenant-scoped text use GET /api/v1/metrics/prometheus.
// @Tags health
// @Produce text/plain
// @Success 200 {string} string
// @Router /metrics [get]
func (h *Handler) prometheusMetrics(c *gin.Context) {
	ctx := c.Request.Context()
	if store.PostgresRLSEnabled() {
		ctx = store.WithRLSBypassContext(ctx)
		c.Request = c.Request.WithContext(ctx)
	}
	text, err := h.alertsFor(c).PrometheusTextWith(alerts.PrometheusOptions{})
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	c.Header("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	c.Header("X-Ash-Metrics-Scope", "global")
	c.String(http.StatusOK, text)
}

// GetMetricsPrometheus godoc
// @Summary Get tenant-scoped Prometheus metrics
// @Description DB-derived Prometheus text for the active space (requires observability:read). Global ops scrape uses GET /metrics.
// @Tags metrics
// @Produce plain
// @Param spaceId query string false "space id (defaults to session space)"
// @Success 200 {string} string "Prometheus text"
// @Failure 403 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/metrics/prometheus [get]
func (h *Handler) getMetricsPrometheus(c *gin.Context) {
	space := firstNonEmptyAPI(c.Query("spaceId"), currentSpace(c))
	if !h.requireRequestSpace(c, space) {
		return
	}
	if !h.requirePermission(c, permObservabilityRead, space) {
		return
	}
	text, err := h.alertsFor(c).PrometheusTextWith(alerts.PrometheusOptions{SpaceID: space})
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	c.Header("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	c.Header("X-Ash-Metrics-Scope", "space:"+space)
	c.String(http.StatusOK, text)
}

// ListAlerts godoc
// @Summary List alert events
// @Tags observability
// @Produce json
// @Param status query string false "active|resolved"
// @Param limit query int false "max items" default(100)
// @Success 200 {object} AlertEventListResponse
// @Failure 403 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/observability/alerts [get]
func (h *Handler) listAlerts(c *gin.Context) {
	space := currentSpace(c)
	if !h.requirePermission(c, permObservabilityRead, space) {
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	rows, err := h.alertsFor(c).ListEvents(space, c.Query("status"), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("ALERT_LIST_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, AlertEventListResponse{Items: rows})
}

// ListAlertRules godoc
// @Summary List alert rules
// @Tags observability
// @Produce json
// @Success 200 {object} AlertRuleListResponse
// @Failure 403 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/observability/alert-rules [get]
func (h *Handler) listAlertRules(c *gin.Context) {
	space := currentSpace(c)
	if !h.requirePermission(c, permObservabilityRead, space) {
		return
	}
	rows, err := h.alertsFor(c).ListRules(space)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("ALERT_RULE_LIST_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, AlertRuleListResponse{Items: rows})
}

// PutAlertRules godoc
// @Summary Update alert rules
// @Tags observability
// @Accept json
// @Produce json
// @Param body body putAlertRulesRequest true "alert rules"
// @Success 200 {object} AlertRuleListResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 403 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/observability/alert-rules [put]
func (h *Handler) putAlertRules(c *gin.Context) {
	space := currentSpace(c)
	if !h.requirePermission(c, permObservabilityWrite, space) {
		return
	}
	var req putAlertRulesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", err.Error()))
		return
	}
	rows, err := h.alertsFor(c).PutRules(space, req.Items)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("ALERT_RULE_UPDATE_FAILED", err.Error()))
		return
	}
	_ = h.dbFor(c).Create(auditRow(space, currentActor(c), "alert.rules_updated", map[string]any{"count": len(req.Items)}))
	c.JSON(http.StatusOK, AlertRuleListResponse{Items: rows})
}

// EvaluateAlerts godoc
// @Summary Evaluate alert rules
// @Tags observability
// @Produce json
// @Success 200 {object} alerts.EvaluationResult
// @Failure 403 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/observability/alerts/evaluate [post]
func (h *Handler) evaluateAlerts(c *gin.Context) {
	space := currentSpace(c)
	if !h.requirePermission(c, permObservabilityRead, space) {
		return
	}
	resp, err := h.alertsFor(c).Evaluate(space)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("ALERT_EVALUATE_FAILED", err.Error()))
		return
	}
	_ = h.dbFor(c).Create(auditRow(space, currentActor(c), "alert.rules_evaluated", map[string]any{
		"results": len(resp.Results), "events": len(resp.Events),
	}))
	c.JSON(http.StatusOK, resp)
}

// GetTrace godoc
// @Summary Get trace-linked records
// @Tags observability
// @Produce json
// @Param traceId path string true "trace id"
// @Success 200 {object} alerts.TraceView
// @Failure 400 {object} APIErrorResponse
// @Failure 403 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/observability/trace/{traceId} [get]
func (h *Handler) getTrace(c *gin.Context) {
	space := currentSpace(c)
	if !h.requirePermission(c, permObservabilityRead, space) {
		return
	}
	traceID := strings.TrimSpace(c.Param("traceId"))
	resp, err := h.alertsFor(c).Trace(space, traceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("TRACE_LOOKUP_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, resp)
}
