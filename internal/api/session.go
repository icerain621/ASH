package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ash-repwiki/ash/internal/session"
)

// CreateAgentSession godoc
// @Summary Create an agent session
// @Description Bind an existing runId or route a goal (optional autoApprove) into a long-lived session document.
// @Tags agents
// @Accept json
// @Produce json
// @Param body body session.CreateRequest true "session create"
// @Success 201 {object} session.View
// @Failure 400 {object} APIErrorResponse
// @Failure 403 {object} APIErrorResponse
// @Router /api/v1/agents/sessions [post]
func (h *Handler) createAgentSession(c *gin.Context) {
	var req session.CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
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
	if req.CreatedBy == "" {
		req.CreatedBy = currentActor(c)
	}
	if req.ActorRole == "" {
		req.ActorRole = currentRole(c)
	}
	view, err := h.sessionFor(c).Create(req)
	if err != nil && view == nil {
		c.JSON(http.StatusBadRequest, errorBody("SESSION_CREATE_FAILED", err.Error()))
		return
	}
	_ = h.dbFor(c).Create(auditRow(req.SpaceID, currentActor(c), "agent.session_created", map[string]any{
		"sessionId": view.ID, "runId": view.RunID, "planId": view.PlanID, "goal": view.Goal,
	})).Error
	c.JSON(http.StatusCreated, view)
}

// GetAgentSession godoc
// @Summary Get agent session
// @Tags agents
// @Produce json
// @Param sessionId path string true "session id"
// @Success 200 {object} session.View
// @Failure 404 {object} APIErrorResponse
// @Router /api/v1/agents/sessions/{sessionId} [get]
func (h *Handler) getAgentSession(c *gin.Context) {
	view, err := h.sessionFor(c).Get(c.Param("sessionId"))
	if err != nil {
		c.JSON(http.StatusNotFound, errorBody("SESSION_NOT_FOUND", err.Error()))
		return
	}
	if !h.requireRequestSpace(c, view.SpaceID) {
		return
	}
	c.JSON(http.StatusOK, view)
}

// PromptAgentSessionTurn godoc
// @Summary Submit a turn.prompt to a session
// @Tags agents
// @Accept json
// @Produce json
// @Param sessionId path string true "session id"
// @Param body body session.TurnRequest true "turn"
// @Success 200 {object} session.View
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Router /api/v1/agents/sessions/{sessionId}/turns [post]
func (h *Handler) promptAgentSessionTurn(c *gin.Context) {
	var req session.TurnRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", err.Error()))
		return
	}
	view, turn, err := h.sessionFor(c).PromptTurn(c.Param("sessionId"), req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, errorBody("SESSION_NOT_FOUND", err.Error()))
			return
		}
		c.JSON(http.StatusBadRequest, errorBody("SESSION_TURN_FAILED", err.Error()))
		return
	}
	if !h.requireRequestSpace(c, view.SpaceID) {
		return
	}
	_ = h.dbFor(c).Create(auditRow(view.SpaceID, currentActor(c), "agent.session_turn", map[string]any{
		"sessionId": view.ID, "turnId": turn.ID, "runId": view.RunID,
	})).Error
	c.JSON(http.StatusOK, view)
}

// ListAgentSessionEvents godoc
// @Summary List session events (bound run events + streamUrl)
// @Tags agents
// @Produce json
// @Param sessionId path string true "session id"
// @Param afterSeq query int false "return events after this seq"
// @Param limit query int false "max items" default(50)
// @Success 200 {object} session.EventsResponse
// @Failure 404 {object} APIErrorResponse
// @Router /api/v1/agents/sessions/{sessionId}/events [get]
func (h *Handler) listAgentSessionEvents(c *gin.Context) {
	afterSeq, _ := strconv.ParseInt(c.DefaultQuery("afterSeq", "0"), 10, 64)
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	out, err := h.sessionFor(c).ListEvents(c.Param("sessionId"), afterSeq, limit)
	if err != nil {
		c.JSON(http.StatusNotFound, errorBody("SESSION_NOT_FOUND", err.Error()))
		return
	}
	view, err := h.sessionFor(c).Get(c.Param("sessionId"))
	if err != nil {
		c.JSON(http.StatusNotFound, errorBody("SESSION_NOT_FOUND", err.Error()))
		return
	}
	if !h.requireRequestSpace(c, view.SpaceID) {
		return
	}
	c.JSON(http.StatusOK, out)
}
