package api

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ash-repwiki/ash/internal/waker"
)

// GetWakerQueue godoc
// @Summary List stale/stuck runs for waker inspection
// @Tags waker
// @Produce json
// @Param spaceId query string false "space id"
// @Param maxAge query string false "duration e.g. 2h"
// @Param limit query int false "max items" default(50)
// @Success 200 {object} waker.QueueResponse
// @Router /api/v1/waker/queue [get]
func (h *Handler) getWakerQueue(c *gin.Context) {
	spaceID := c.DefaultQuery("spaceId", currentSpace(c))
	if !h.requireTargetSpace(c, spaceID) {
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	out, err := h.wakerFor(c).Queue(spaceID, c.Query("maxAge"), limit)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("WAKER_QUEUE_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, out)
}

// PostWakerSweep godoc
// @Summary Sweep stale runs (report/flag/cancel with safety gates)
// @Tags waker
// @Accept json
// @Produce json
// @Param body body waker.SweepRequest true "sweep"
// @Success 200 {object} waker.SweepResponse
// @Failure 400 {object} APIErrorResponse
// @Router /api/v1/waker/sweep [post]
func (h *Handler) postWakerSweep(c *gin.Context) {
	var req waker.SweepRequest
	if err := c.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", err.Error()))
		return
	}
	req.SpaceID = firstNonEmptyAPI(strings.TrimSpace(req.SpaceID), currentSpace(c))
	if !h.requireTargetSpace(c, req.SpaceID) {
		return
	}
	if !h.requirePermission(c, permRunCreate, req.SpaceID) {
		return
	}
	if req.ActorID == "" {
		req.ActorID = currentActor(c)
	}
	resp, err := h.wakerFor(c).Sweep(req)
	if err != nil {
		if errors.Is(err, waker.ErrCancelDenied) {
			c.JSON(http.StatusBadRequest, errorBody("WAKER_CANCEL_DENIED", err.Error()))
			return
		}
		c.JSON(http.StatusBadRequest, errorBody("WAKER_SWEEP_FAILED", err.Error()))
		return
	}
	eventType := "waker.sweep_completed"
	if strings.EqualFold(resp.Action, "cancel") && resp.Canceled > 0 {
		eventType = "waker.cancel_completed"
	}
	_ = h.dbFor(c).Create(auditRow(req.SpaceID, req.ActorID, eventType, map[string]any{
		"matched": resp.Matched, "flagged": resp.Flagged, "canceled": resp.Canceled,
		"dryRun": resp.DryRun, "action": resp.Action,
		"maxAge": resp.MaxAge, "runIds": resp.RunIDs, "summary": resp.Summary,
	})).Error
	c.JSON(http.StatusOK, resp)
}

// GetWakerStatus godoc
// @Summary Waker ticker config, enabled duties, and recent duty runs
// @Tags waker
// @Produce json
// @Param spaceId query string false "space id"
// @Param recent query int false "max recent duty_run rows" default(10)
// @Success 200 {object} waker.StatusResponse
// @Failure 400 {object} APIErrorResponse
// @Router /api/v1/waker/status [get]
func (h *Handler) getWakerStatus(c *gin.Context) {
	spaceID := c.DefaultQuery("spaceId", currentSpace(c))
	if !h.requireTargetSpace(c, spaceID) {
		return
	}
	recent, _ := strconv.Atoi(c.DefaultQuery("recent", "10"))
	out, err := h.wakerFor(c).Status(spaceID, recent)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("WAKER_STATUS_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, out)
}

type wakerDutiesResponse struct {
	Duties []waker.DutyStatusView `json:"duties"`
}

// GetWakerDuties godoc
// @Summary List waker duties for a space
// @Tags waker
// @Produce json
// @Param spaceId query string false "space id"
// @Success 200 {object} wakerDutiesResponse
// @Failure 400 {object} APIErrorResponse
// @Router /api/v1/waker/duties [get]
func (h *Handler) getWakerDuties(c *gin.Context) {
	spaceID := c.DefaultQuery("spaceId", currentSpace(c))
	if !h.requireTargetSpace(c, spaceID) {
		return
	}
	duties, err := h.wakerFor(c).ListDuties(spaceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("WAKER_DUTIES_FAILED", err.Error()))
		return
	}
	views := make([]waker.DutyStatusView, 0, len(duties))
	for _, d := range duties {
		views = append(views, waker.DutyStatusView{
			ID: d.ID, SpaceID: d.SpaceID, Kind: d.Kind, Enabled: d.Enabled,
			IntervalMs: d.IntervalMs, NextRunAt: d.NextRunAt, UpdatedAt: d.UpdatedAt,
		})
	}
	c.JSON(http.StatusOK, wakerDutiesResponse{Duties: views})
}

type wakerDutyRunRequest struct {
	DryRun *bool `json:"dryRun,omitempty"`
}

// PostWakerDutyRun godoc
// @Summary Force one report pass for a duty (dryRun default true; never cancel)
// @Tags waker
// @Accept json
// @Produce json
// @Param id path string true "duty id"
// @Param spaceId query string false "space id"
// @Param body body wakerDutyRunRequest false "run; dryRun defaults to true"
// @Success 200 {object} waker.SweepResponse
// @Failure 400 {object} APIErrorResponse
// @Router /api/v1/waker/duties/{id}/run [post]
func (h *Handler) postWakerDutyRun(c *gin.Context) {
	spaceID := c.DefaultQuery("spaceId", currentSpace(c))
	if !h.requireTargetSpace(c, spaceID) {
		return
	}
	if !h.requirePermission(c, permRunCreate, spaceID) {
		return
	}
	var req wakerDutyRunRequest
	if err := c.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", err.Error()))
		return
	}
	dryRun := true
	if req.DryRun != nil {
		dryRun = *req.DryRun
	}
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", "duty id is required"))
		return
	}
	duties, err := h.wakerFor(c).ListDuties(spaceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("WAKER_DUTY_RUN_FAILED", err.Error()))
		return
	}
	found := false
	for _, d := range duties {
		if d.ID == id {
			found = true
			break
		}
	}
	if !found {
		c.JSON(http.StatusBadRequest, errorBody("WAKER_DUTY_RUN_FAILED", "duty not found"))
		return
	}
	resp, err := h.wakerFor(c).RunDuty(id, dryRun)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("WAKER_DUTY_RUN_FAILED", err.Error()))
		return
	}
	actorID := currentActor(c)
	_ = h.dbFor(c).Create(auditRow(spaceID, actorID, "waker.duty_completed", map[string]any{
		"dutyId": id, "matched": resp.Matched, "flagged": resp.Flagged,
		"canceled": resp.Canceled, "dryRun": resp.DryRun, "action": resp.Action,
		"summary": resp.Summary, "runIds": resp.RunIDs,
	})).Error
	c.JSON(http.StatusOK, resp)
}

type wakerDutyEnableRequest struct {
	Enabled bool `json:"enabled"`
}

// PostWakerDutyEnable godoc
// @Summary Enable or disable a waker duty
// @Tags waker
// @Accept json
// @Produce json
// @Param id path string true "duty id"
// @Param spaceId query string false "space id"
// @Param body body wakerDutyEnableRequest true "enabled"
// @Success 200 {object} waker.DutyStatusView
// @Failure 400 {object} APIErrorResponse
// @Router /api/v1/waker/duties/{id}/enable [post]
func (h *Handler) postWakerDutyEnable(c *gin.Context) {
	spaceID := c.DefaultQuery("spaceId", currentSpace(c))
	if !h.requireTargetSpace(c, spaceID) {
		return
	}
	if !h.requirePermission(c, permRunCreate, spaceID) {
		return
	}
	var req wakerDutyEnableRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", err.Error()))
		return
	}
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", "duty id is required"))
		return
	}
	duties, err := h.wakerFor(c).ListDuties(spaceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("WAKER_DUTY_ENABLE_FAILED", err.Error()))
		return
	}
	found := false
	for _, d := range duties {
		if d.ID == id {
			found = true
			break
		}
	}
	if !found {
		c.JSON(http.StatusBadRequest, errorBody("WAKER_DUTY_ENABLE_FAILED", "duty not found"))
		return
	}
	duty, err := h.wakerFor(c).SetDutyEnabled(id, req.Enabled)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("WAKER_DUTY_ENABLE_FAILED", err.Error()))
		return
	}
	actorID := currentActor(c)
	_ = h.dbFor(c).Create(auditRow(spaceID, actorID, "waker.duty_enabled", map[string]any{
		"dutyId": id, "kind": duty.Kind, "enabled": req.Enabled,
	})).Error
	c.JSON(http.StatusOK, waker.DutyStatusView{
		ID: duty.ID, SpaceID: duty.SpaceID, Kind: duty.Kind, Enabled: duty.Enabled,
		IntervalMs: duty.IntervalMs, NextRunAt: duty.NextRunAt, UpdatedAt: duty.UpdatedAt,
	})
}

func (h *Handler) wakerFor(c *gin.Context) *waker.Service {
	if h == nil || h.waker == nil {
		return h.waker
	}
	return h.waker.WithContext(c.Request.Context())
}
