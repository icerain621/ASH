package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ash-repwiki/ash/internal/runs"
)

// SpawnSubRun godoc
// @Summary Spawn a child run under a parent
// @Tags runs
// @Accept json
// @Produce json
// @Param runId path string true "parent run id"
// @Param body body runs.SpawnRequest true "spawn request"
// @Success 201 {object} runs.CreateResponse
// @Router /api/v1/runs/{runId}/sub-runs [post]
func (h *Handler) spawnSubRun(c *gin.Context) {
	parentID := c.Param("runId")
	if !h.requireRunAccess(c, parentID) {
		return
	}
	var req runs.SpawnRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", err.Error()))
		return
	}
	if strings.TrimSpace(req.ActorRole) == "" {
		req.ActorRole = currentRole(c)
	}
	resp, err := h.runsFor(c).Spawn(parentID, req)
	if resp == nil {
		code := http.StatusBadRequest
		msg := err.Error()
		if strings.Contains(msg, "not found") {
			code = http.StatusNotFound
		}
		c.JSON(code, errorBody("SUBRUN_SPAWN_FAILED", msg))
		return
	}
	c.Header("X-Run-Id", resp.RunID)
	c.Header("X-Trace-Id", resp.TraceID)
	out := RunCreateResponse{RunID: resp.RunID, TraceID: resp.TraceID}
	if err != nil {
		out.ExecutionError = err.Error()
		if sum, getErr := h.runsFor(c).Get(resp.RunID); getErr == nil {
			out.Status = sum.Status
		} else {
			out.Status = "failed"
		}
	}
	c.JSON(http.StatusCreated, out)
}

// GetRunTree godoc
// @Summary Get spawn tree for a run (rooted at rootRunId)
// @Tags runs
// @Produce json
// @Param runId path string true "run id"
// @Success 200 {object} runs.TreeResponse
// @Router /api/v1/runs/{runId}/tree [get]
func (h *Handler) getRunTree(c *gin.Context) {
	runID := c.Param("runId")
	if !h.requireRunAccess(c, runID) {
		return
	}
	tree, err := h.runsFor(c).Tree(runID)
	if err != nil {
		c.JSON(http.StatusNotFound, errorBody("SUBRUN_TREE_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, tree)
}
