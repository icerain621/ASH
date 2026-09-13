package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ash-repwiki/ash/internal/interaction"
)

// ListInteractionSessionThreads godoc
// @Summary List threads for a session
// @Tags interactions
// @Produce json
// @Param sessionId path string true "session id"
// @Success 200 {object} interaction.SessionThreadsView
// @Failure 400 {object} APIErrorResponse
// @Router /api/v1/interactions/sessions/{sessionId}/threads [get]
func (h *Handler) listInteractionSessionThreads(c *gin.Context) {
	sessionID := strings.TrimSpace(c.Param("sessionId"))
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", "sessionId is required"))
		return
	}
	items, err := h.interactionFor(c).ListThreads(sessionID)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", err.Error()))
		return
	}
	if len(items) > 0 && !h.requireRequestSpace(c, items[0].SpaceID) {
		return
	}
	c.JSON(http.StatusOK, interaction.SessionThreadsView{SessionID: sessionID, Items: items})
}

// GetInteractionByRun godoc
// @Summary Resolve Session/Thread for a run
// @Tags interactions
// @Produce json
// @Param runId path string true "run id"
// @Success 200 {object} interaction.ByRunView
// @Failure 404 {object} APIErrorResponse
// @Router /api/v1/interactions/by-run/{runId} [get]
func (h *Handler) getInteractionByRun(c *gin.Context) {
	runID := c.Param("runId")
	if !h.requireRunAccess(c, runID) {
		return
	}
	out, err := h.interactionFor(c).ByRun(runID)
	if err != nil {
		c.JSON(http.StatusNotFound, errorBody("INTERACTION_NOT_FOUND", err.Error()))
		return
	}
	if !h.requireRequestSpace(c, out.Thread.SpaceID) {
		return
	}
	c.JSON(http.StatusOK, out)
}

// GetInteractionThread godoc
// @Summary Fold thread timeline nodes + digest
// @Tags interactions
// @Produce json
// @Param threadId path string true "thread id"
// @Success 200 {object} interaction.FoldResult
// @Failure 404 {object} APIErrorResponse
// @Router /api/v1/interactions/threads/{threadId} [get]
func (h *Handler) getInteractionThread(c *gin.Context) {
	fold, err := h.interactionFor(c).FoldThread(c.Param("threadId"))
	if err != nil {
		c.JSON(http.StatusNotFound, errorBody("THREAD_NOT_FOUND", err.Error()))
		return
	}
	if !h.requireRequestSpace(c, fold.SpaceID) {
		return
	}
	c.JSON(http.StatusOK, fold)
}

// ListInteractionMemoryLinks godoc
// @Summary Memory links for a thread
// @Tags interactions
// @Produce json
// @Param threadId path string true "thread id"
// @Success 200 {object} map[string]any
// @Failure 404 {object} APIErrorResponse
// @Router /api/v1/interactions/threads/{threadId}/memory-links [get]
func (h *Handler) listInteractionMemoryLinks(c *gin.Context) {
	links, err := h.interactionFor(c).ListMemoryLinks(c.Param("threadId"))
	if err != nil {
		c.JSON(http.StatusNotFound, errorBody("THREAD_NOT_FOUND", err.Error()))
		return
	}
	th, err := h.interactionFor(c).GetThread(c.Param("threadId"))
	if err != nil {
		c.JSON(http.StatusNotFound, errorBody("THREAD_NOT_FOUND", err.Error()))
		return
	}
	if !h.requireRequestSpace(c, th.SpaceID) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"threadId": th.ID, "items": links})
}

// EnsureInteractionThread godoc
// @Summary Ensure main thread for a run
// @Tags interactions
// @Accept json
// @Produce json
// @Param body body interaction.EnsureRequest true "ensure"
// @Success 200 {object} interaction.Thread
// @Failure 400 {object} APIErrorResponse
// @Router /api/v1/interactions/threads/ensure [post]
func (h *Handler) ensureInteractionThread(c *gin.Context) {
	var req interaction.EnsureRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", err.Error()))
		return
	}
	if strings.TrimSpace(req.RunID) == "" {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", "runId is required"))
		return
	}
	if !h.requireRunAccess(c, req.RunID) {
		return
	}
	if req.SpaceID == "" {
		req.SpaceID = currentSpace(c)
	}
	th, _, err := h.interactionFor(c).EnsureThread(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("THREAD_ENSURE_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, th)
}

// SealInteractionThread godoc
// @Summary Seal thread digest
// @Tags interactions
// @Produce json
// @Param threadId path string true "thread id"
// @Success 200 {object} interaction.Thread
// @Failure 404 {object} APIErrorResponse
// @Router /api/v1/interactions/threads/{threadId}/seal [post]
func (h *Handler) sealInteractionThread(c *gin.Context) {
	th, err := h.interactionFor(c).GetThread(c.Param("threadId"))
	if err != nil {
		c.JSON(http.StatusNotFound, errorBody("THREAD_NOT_FOUND", err.Error()))
		return
	}
	if !h.requireRequestSpace(c, th.SpaceID) {
		return
	}
	sealed, err := h.interactionFor(c).Seal(th.ID)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("THREAD_SEAL_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, sealed)
}

// ReplayInteractionThread godoc
// @Summary Replay fold and verify sealed digest
// @Tags interactions
// @Produce json
// @Param threadId path string true "thread id"
// @Success 200 {object} interaction.ReplayResult
// @Failure 404 {object} APIErrorResponse
// @Failure 409 {object} APIErrorResponse
// @Router /api/v1/interactions/threads/{threadId}/replay [post]
func (h *Handler) replayInteractionThread(c *gin.Context) {
	th, err := h.interactionFor(c).GetThread(c.Param("threadId"))
	if err != nil {
		c.JSON(http.StatusNotFound, errorBody("THREAD_NOT_FOUND", err.Error()))
		return
	}
	if !h.requireRequestSpace(c, th.SpaceID) {
		return
	}
	out, err := h.interactionFor(c).Replay(th.ID)
	if err != nil {
		if errors.Is(err, interaction.ErrReplayDigestMismatch) {
			body := errorBody("REPLAY_DIGEST_MISMATCH", err.Error())
			body["data"] = out
			c.JSON(http.StatusConflict, body)
			return
		}
		c.JSON(http.StatusBadRequest, errorBody("THREAD_REPLAY_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, out)
}

type compareThreadsRequest struct {
	Left  string `json:"left"`
	Right string `json:"right"`
}

// CompareInteractionThreads godoc
// @Summary Compare two threads
// @Tags interactions
// @Accept json
// @Produce json
// @Param body body compareThreadsRequest true "left/right thread ids"
// @Success 200 {object} interaction.CompareResult
// @Failure 400 {object} APIErrorResponse
// @Router /api/v1/interactions/compare [post]
func (h *Handler) compareInteractionThreads(c *gin.Context) {
	var req compareThreadsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", err.Error()))
		return
	}
	left, err := h.interactionFor(c).GetThread(req.Left)
	if err != nil {
		c.JSON(http.StatusNotFound, errorBody("THREAD_NOT_FOUND", err.Error()))
		return
	}
	right, err := h.interactionFor(c).GetThread(req.Right)
	if err != nil {
		c.JSON(http.StatusNotFound, errorBody("THREAD_NOT_FOUND", err.Error()))
		return
	}
	if !h.requireRequestSpace(c, left.SpaceID) || !h.requireRequestSpace(c, right.SpaceID) {
		return
	}
	out, err := h.interactionFor(c).Compare(req.Left, req.Right)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("THREAD_COMPARE_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, out)
}
