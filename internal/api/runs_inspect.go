package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// GetRunTimeline godoc
// @Summary Get run timeline
// @Tags runs
// @Produce json
// @Param runId path string true "run id"
// @Param limit query int false "max items" default(500)
// @Success 200 {object} TimelineAPIResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/runs/{runId}/timeline [get]
func (h *Handler) getRunTimeline(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "500"))
	runID := c.Param("runId")
	if !h.requireRunAccess(c, runID) {
		return
	}
	resp, err := h.runs.Timeline(runID, limit)
	if err != nil {
		c.JSON(http.StatusNotFound, errorBody("RUN_TIMELINE_NOT_FOUND", err.Error()))
		return
	}
	c.JSON(http.StatusOK, resp)
}

// ListRunToolCalls godoc
// @Summary List run tool calls
// @Tags runs
// @Produce json
// @Param runId path string true "run id"
// @Success 200 {object} ToolCallListResponse
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/runs/{runId}/tool-calls [get]
func (h *Handler) listRunToolCalls(c *gin.Context) {
	runID := c.Param("runId")
	if !h.requireRunAccess(c, runID) {
		return
	}
	items, err := h.runs.ToolCalls(runID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("TOOL_CALL_LIST_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, ToolCallListResponse{Items: items})
}

// ListRunAgentTasks godoc
// @Summary List run agent tasks
// @Tags runs
// @Produce json
// @Param runId path string true "run id"
// @Success 200 {object} AgentTaskListResponse
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/runs/{runId}/agent-tasks [get]
func (h *Handler) listRunAgentTasks(c *gin.Context) {
	runID := c.Param("runId")
	if !h.requireRunAccess(c, runID) {
		return
	}
	items, err := h.runs.AgentTasks(runID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("AGENT_TASK_LIST_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, AgentTaskListResponse{Items: items})
}

// ListRunQualityMetrics godoc
// @Summary List run quality metrics
// @Tags runs
// @Produce json
// @Param runId path string true "run id"
// @Success 200 {object} QualityMetricListResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/runs/{runId}/quality-metrics [get]
func (h *Handler) listRunQualityMetrics(c *gin.Context) {
	runID := c.Param("runId")
	if !h.requireRunAccess(c, runID) {
		return
	}
	items, err := h.runs.QualityMetrics(runID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("QUALITY_METRIC_LIST_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, QualityMetricListResponse{Items: items})
}
