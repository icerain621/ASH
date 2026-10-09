package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ash-repwiki/ash/internal/observability/lenses"
)

// GetObserveGlobalLens godoc
// @Summary Global observation lens snapshot
// @Tags observability
// @Produce json
// @Param spaceId query string false "space id"
// @Success 200 {object} lenses.GlobalView
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/observability/lenses/global [get]
func (h *Handler) getObserveGlobalLens(c *gin.Context) {
	space := strings.TrimSpace(c.Query("spaceId"))
	if space == "" {
		space = currentSpace(c)
	}
	view, err := lenses.BuildGlobal(h.db.BindContext(c.Request.Context()), space)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("OBSERVE_GLOBAL_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, view)
}

// GetObserveAgentLens godoc
// @Summary Agent observation lens for one run
// @Tags observability
// @Produce json
// @Param runId query string true "run id"
// @Success 200 {object} lenses.AgentView
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Router /api/v1/observability/lenses/agent [get]
func (h *Handler) getObserveAgentLens(c *gin.Context) {
	runID := strings.TrimSpace(c.Query("runId"))
	if runID == "" {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", "runId is required"))
		return
	}
	if !h.requireRunAccess(c, runID) {
		return
	}
	view, err := lenses.BuildAgent(h.db.BindContext(c.Request.Context()), runID)
	if err != nil {
		c.JSON(http.StatusNotFound, errorBody("OBSERVE_AGENT_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, view)
}

// GetObserveMemoryLens godoc
// @Summary Memory lineage observation lens
// @Tags observability
// @Produce json
// @Param id query string true "memory id"
// @Success 200 {object} lenses.MemoryLineageView
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Router /api/v1/observability/lenses/memory [get]
func (h *Handler) getObserveMemoryLens(c *gin.Context) {
	id := strings.TrimSpace(c.Query("id"))
	if id == "" {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", "id is required"))
		return
	}
	view, err := lenses.BuildMemoryLineage(h.db.BindContext(c.Request.Context()), id)
	if err != nil {
		c.JSON(http.StatusNotFound, errorBody("OBSERVE_MEMORY_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, view)
}
