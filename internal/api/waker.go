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

func (h *Handler) wakerFor(c *gin.Context) *waker.Service {
	if h == nil || h.waker == nil {
		return h.waker
	}
	return h.waker.WithContext(c.Request.Context())
}
