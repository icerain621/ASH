package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ash-repwiki/ash/internal/diffreview"
	"github.com/ash-repwiki/ash/internal/events"
	"github.com/ash-repwiki/ash/internal/quest"
	"github.com/ash-repwiki/ash/internal/store"
)

// Ensure swag/openapi can resolve quest types.
var _ = quest.BoardResponse{}
var _ = diffreview.DiffView{}

type rateStepRequest struct {
	Rating  int    `json:"rating" binding:"required"`
	Comment string `json:"comment"`
	ActorID string `json:"actorId"`
}

type diffCommentListResponse struct {
	Items []diffreview.CommentView `json:"items"`
}

// QuestBoard godoc
// @Summary Quest workbench kanban board
// @Tags quest
// @Produce json
// @Param limit query int false "max items" default(80)
// @Success 200 {object} quest.BoardResponse
// @Router /api/v1/quest/board [get]
func (h *Handler) questBoard(c *gin.Context) {
	space := currentSpace(c)
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "80"))
	out, err := h.quest.WithContext(c.Request.Context()).Board(space, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("QUEST_BOARD_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, out)
}

// QuestStream godoc
// @Summary Quest board live SSE (plan.* / run status)
// @Description Space-scoped event stream for Quest kanban. Fresh connect starts from now (live-only). Resume via Last-Event-ID.
// @Tags quest
// @Produce text/event-stream
// @Param Last-Event-ID header string false "resume after this event id"
// @Param lastEventId query string false "resume after this event id"
// @Success 200 {string} string "SSE stream"
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/quest/stream [get]
func (h *Handler) questStream(c *gin.Context) {
	space := currentSpace(c)

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")

	afterTS := time.Now().UTC().UnixMilli()
	lastID := strings.TrimSpace(c.GetHeader("Last-Event-ID"))
	if lastID == "" {
		lastID = strings.TrimSpace(c.Query("Last-Event-ID"))
	}
	if lastID == "" {
		lastID = strings.TrimSpace(c.Query("lastEventId"))
	}
	if lastID != "" {
		if ts, err := h.eventsFor(c).TSFromEventID(lastID); err == nil {
			afterTS = ts
		}
	}

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, errorBody("SSE_UNSUPPORTED", "streaming not supported"))
		return
	}

	writeEvents := func(evs []events.Envelope) {
		for _, ev := range evs {
			data, _ := json.Marshal(ev)
			_, _ = c.Writer.Write([]byte("id: " + ev.ID + "\n"))
			_, _ = c.Writer.Write([]byte("event: " + ev.Type + "\n"))
			_, _ = c.Writer.Write([]byte("data: "))
			_, _ = c.Writer.Write(data)
			_, _ = c.Writer.Write([]byte("\n\n"))
			flusher.Flush()
			if ev.TS > afterTS {
				afterTS = ev.TS
			}
		}
	}

	opened := false
	failed := false
	defer func() {
		if opened && !failed {
			h.auditStream(c, space, "stream.session_closed", map[string]any{
				"spaceId": space, "channel": "quest", "reason": "client_disconnect",
			})
		}
	}()

	// Comment so proxies keep the connection; no backlog on fresh connect.
	_, _ = c.Writer.Write([]byte(": quest stream open\n\n"))
	flusher.Flush()
	opened = true
	h.auditStream(c, space, "stream.session_opened", map[string]any{
		"spaceId": space, "channel": "quest", "afterTS": afterTS,
	})

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-ticker.C:
			evs, err := h.eventsFor(c).ListAfterSpace(space, afterTS, 100)
			if err != nil {
				failed = true
				h.auditStream(c, space, "stream.session_failed", map[string]any{
					"spaceId": space, "channel": "quest", "reason": "event_poll_failed", "error": err.Error(),
				})
				return
			}
			if len(evs) == 0 {
				continue
			}
			writeEvents(evs)
		}
	}
}

// GetRunDiff godoc
// @Summary Get parsed unified diff for a run
// @Tags quest
// @Produce json
// @Param runId path string true "run id"
// @Success 200 {object} diffreview.DiffView
// @Router /api/v1/runs/{runId}/diff [get]
func (h *Handler) getRunDiff(c *gin.Context) {
	runID := c.Param("runId")
	if !h.requireRunAccess(c, runID) {
		return
	}
	view, err := h.diffReview.WithContext(c.Request.Context()).GetDiff(runID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, errorBody("RUN_NOT_FOUND", "run not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, errorBody("DIFF_GET_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, view)
}

// RejectRunDiff godoc
// @Summary Reject a file or the full run diff (Quest review)
// @Tags quest
// @Accept json
// @Produce json
// @Param runId path string true "run id"
// @Param body body diffreview.RejectRequest true "reject request"
// @Success 200 {object} diffreview.RejectResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/runs/{runId}/diff/reject [post]
func (h *Handler) rejectRunDiff(c *gin.Context) {
	runID := c.Param("runId")
	if !h.requireRunAccess(c, runID) {
		return
	}
	space := currentSpace(c)
	if !h.requirePermission(c, permRunApprove, space) {
		return
	}
	var req diffreview.RejectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", err.Error()))
		return
	}
	if req.ActorID == "" {
		req.ActorID = currentActor(c)
	}
	resp, err := h.diffReview.WithContext(c.Request.Context()).Reject(runID, req)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, errorBody("RUN_NOT_FOUND", err.Error()))
			return
		}
		c.JSON(http.StatusBadRequest, errorBody("DIFF_REJECT_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, resp)
}

// ListDiffComments godoc
// @Summary List line-level diff review comments
// @Tags quest
// @Produce json
// @Param runId path string true "run id"
// @Success 200 {object} diffCommentListResponse
// @Router /api/v1/runs/{runId}/diff/comments [get]
func (h *Handler) listDiffComments(c *gin.Context) {
	runID := c.Param("runId")
	if !h.requireRunAccess(c, runID) {
		return
	}
	items, err := h.diffReview.WithContext(c.Request.Context()).ListComments(runID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("DIFF_COMMENT_LIST_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, diffCommentListResponse{Items: items})
}

// CreateDiffComment godoc
// @Summary Create a line-level diff review comment
// @Tags quest
// @Accept json
// @Produce json
// @Param runId path string true "run id"
// @Param body body diffreview.CreateCommentRequest true "comment"
// @Success 201 {object} diffreview.CommentView
// @Router /api/v1/runs/{runId}/diff/comments [post]
func (h *Handler) createDiffComment(c *gin.Context) {
	runID := c.Param("runId")
	if !h.requireRunAccess(c, runID) {
		return
	}
	space := currentSpace(c)
	if !h.requirePermission(c, permRunCreate, space) {
		return
	}
	var req diffreview.CreateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", err.Error()))
		return
	}
	if req.CreatedBy == "" {
		req.CreatedBy = currentActor(c)
	}
	view, err := h.diffReview.WithContext(c.Request.Context()).CreateComment(runID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("DIFF_COMMENT_CREATE_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusCreated, view)
}

// RateRunStep godoc
// @Summary Rate a run step (feedback targetType=run_step)
// @Tags quest
// @Accept json
// @Produce json
// @Param runId path string true "run id"
// @Param stepId path string true "step id"
// @Param body body rateStepRequest true "rating"
// @Success 201 {object} store.Feedback
// @Router /api/v1/runs/{runId}/steps/{stepId}/rate [post]
func (h *Handler) rateRunStep(c *gin.Context) {
	runID := c.Param("runId")
	stepID := strings.TrimSpace(c.Param("stepId"))
	if stepID == "" {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", "stepId required"))
		return
	}
	if !h.requireRunAccess(c, runID) {
		return
	}
	var req rateStepRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", err.Error()))
		return
	}
	if req.Rating < 1 || req.Rating > 5 {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", "rating must be 1..5"))
		return
	}
	space := currentSpace(c)
	if !h.requirePermission(c, permFeedbackWrite, space) {
		return
	}
	now := time.Now().UTC()
	row := store.Feedback{
		ID: "fb_" + uuid.NewString(), SpaceID: space,
		TargetType: "run_step", TargetID: stepID, RunID: runID, Rating: req.Rating,
		Category: "quality", Status: "open", Severity: "info", Source: "quest",
		Comment: req.Comment, ActorID: firstNonEmptyAPI(req.ActorID, currentActor(c)),
		CreatedAt: now, UpdatedAt: now,
	}
	if req.Rating <= 2 {
		row.Severity = "warn"
	}
	if err := h.dbFor(c).Create(&row).Error; err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("FEEDBACK_CREATE_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusCreated, row)
}
