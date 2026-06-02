package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ash-repwiki/ash/internal/runs"
)

// ResumeRun godoc
// @Summary Resume a failed run
// @Description Re-executes a failed run on the same run id (M0: full re-run).
// @Tags runs
// @Produce json
// @Param runId path string true "run id"
// @Success 200 {object} runs.ResumeResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 409 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/runs/{runId}/resume [post]
func (h *Handler) resumeRun(c *gin.Context) {
	runID := c.Param("runId")
	resp, err := h.runs.Resume(runID)
	if err != nil {
		writeRunControlError(c, err)
		return
	}
	c.Header("X-Run-Id", resp.RunID)
	c.Header("X-Trace-Id", resp.TraceID)
	c.JSON(http.StatusOK, resp)
}

// ReplayRun godoc
// @Summary Replay a run
// @Description Creates a new run from an existing one (exact or latest_memory mode).
// @Tags runs
// @Accept json
// @Produce json
// @Param runId path string true "source run id"
// @Param body body runs.ReplayRequest true "replay request"
// @Success 201 {object} runs.ReplayResponse
// @Header 201 {string} X-Run-Id "new run id"
// @Header 201 {string} X-Trace-Id "new trace id"
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/runs/{runId}/replay [post]
func (h *Handler) replayRun(c *gin.Context) {
	runID := c.Param("runId")
	var req runs.ReplayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", err.Error()))
		return
	}
	resp, err := h.runs.Replay(runID, req)
	if err != nil {
		writeRunControlError(c, err)
		return
	}
	c.Header("X-Run-Id", resp.RunID)
	c.Header("X-Trace-Id", resp.TraceID)
	c.JSON(http.StatusCreated, resp)
}

func writeRunControlError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, runs.ErrRunNotFound):
		c.JSON(http.StatusNotFound, errorBody("RUN_NOT_FOUND", err.Error()))
	case errors.Is(err, runs.ErrRunNotResumable):
		c.JSON(http.StatusConflict, errorBody("RUN_NOT_RESUMABLE", err.Error()))
	case errors.Is(err, runs.ErrRunMetaMissing):
		c.JSON(http.StatusConflict, errorBody("RUN_META_MISSING", err.Error()))
	case errors.Is(err, runs.ErrInvalidReplayMode):
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REPLAY_MODE", err.Error()))
	default:
		c.JSON(http.StatusInternalServerError, errorBody("RUN_CONTROL_FAILED", err.Error()))
	}
}
