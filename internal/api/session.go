package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ash-repwiki/ash/internal/events"
	"github.com/ash-repwiki/ash/internal/session"
)

// ListAgentSessions godoc
// @Summary List agent sessions
// @Description List agent.session audit documents for a space, newest updatedAt first. Closed sessions are excluded unless includeClosed=1.
// @Tags agents
// @Produce json
// @Param spaceId query string false "space id (default: current space)"
// @Param limit query int false "max items" default(50)
// @Param includeClosed query string false "set to 1/true to include status=closed"
// @Success 200 {object} session.ListResponse
// @Failure 403 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/agents/sessions [get]
func (h *Handler) listAgentSessions(c *gin.Context) {
	spaceID := firstNonEmptyAPI(strings.TrimSpace(c.Query("spaceId")), currentSpace(c))
	if !h.requireRequestSpace(c, spaceID) {
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	includeClosed := queryTruthy(c.Query("includeClosed"))
	items, err := h.sessionFor(c).List(spaceID, limit, includeClosed)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("SESSION_LIST_FAILED", err.Error()))
		return
	}
	if items == nil {
		items = []session.View{}
	}
	for i := range items {
		h.enrichSessionWorkspace(c, &items[i])
	}
	c.JSON(http.StatusOK, session.ListResponse{Items: items})
}

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
	if wsID := strings.TrimSpace(req.WorkspaceID); wsID != "" {
		if _, attachErr := h.agentWorkspaceFor(c).Attach(wsID, view.ID); attachErr != nil {
			c.JSON(http.StatusBadRequest, errorBody("SESSION_WORKSPACE_ATTACH_FAILED", attachErr.Error()))
			return
		}
		view.WorkspaceID = wsID
		if saved, setErr := h.sessionFor(c).SetWorkspaceID(view.ID, wsID); setErr == nil {
			view = saved
		}
	}
	_ = h.dbFor(c).Create(auditRow(req.SpaceID, currentActor(c), "agent.session_created", map[string]any{
		"sessionId": view.ID, "runId": view.RunID, "planId": view.PlanID, "goal": view.Goal,
		"providerKind": view.ProviderKind, "providerAdapter": view.ProviderAdapter,
		"providerFallback": view.ProviderFallback, "workspaceId": view.WorkspaceID,
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
	h.enrichSessionWorkspace(c, view)
	c.JSON(http.StatusOK, view)
}

// PatchAgentSession godoc
// @Summary Patch agent session seats
// @Description Partial update: title, providerKind, planId, permissionMode.
// @Tags agents
// @Accept json
// @Produce json
// @Param sessionId path string true "session id"
// @Param body body session.PatchRequest true "patch"
// @Success 200 {object} session.View
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Router /api/v1/agents/sessions/{sessionId} [patch]
func (h *Handler) patchAgentSession(c *gin.Context) {
	var req session.PatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", err.Error()))
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
	if !h.requirePermission(c, permRunCreate, view.SpaceID) {
		return
	}
	updated, err := h.sessionFor(c).Update(c.Param("sessionId"), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("SESSION_UPDATE_FAILED", err.Error()))
		return
	}
	_ = h.dbFor(c).Create(auditRow(updated.SpaceID, currentActor(c), "agent.session_updated", map[string]any{
		"sessionId": updated.ID, "title": updated.Title, "runId": updated.RunID,
		"providerKind": updated.ProviderKind, "planId": updated.PlanID,
		"permissionMode": updated.PermissionMode,
	})).Error
	c.JSON(http.StatusOK, updated)
}

// CloseAgentSession godoc
// @Summary Soft-close agent session
// @Description Sets status=closed (soft delete). List excludes closed by default.
// @Tags agents
// @Produce json
// @Param sessionId path string true "session id"
// @Success 200 {object} session.View
// @Failure 404 {object} APIErrorResponse
// @Router /api/v1/agents/sessions/{sessionId} [delete]
func (h *Handler) closeAgentSession(c *gin.Context) {
	view, err := h.sessionFor(c).Get(c.Param("sessionId"))
	if err != nil {
		c.JSON(http.StatusNotFound, errorBody("SESSION_NOT_FOUND", err.Error()))
		return
	}
	if !h.requireRequestSpace(c, view.SpaceID) {
		return
	}
	if !h.requirePermission(c, permRunCreate, view.SpaceID) {
		return
	}
	closed, err := h.sessionFor(c).Close(c.Param("sessionId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("SESSION_CLOSE_FAILED", err.Error()))
		return
	}
	_ = h.dbFor(c).Create(auditRow(closed.SpaceID, currentActor(c), "agent.session_closed", map[string]any{
		"sessionId": closed.ID, "runId": closed.RunID, "status": closed.Status,
	})).Error
	c.JSON(http.StatusOK, closed)
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

// AgentSessionIntent godoc
// @Summary Apply a thin session intent (prompt|approve|cancel|stop|reject|command)
// @Description Fail-closed: approve without an approvable run gate returns 409. action "stop" is an alias of "cancel". Unknown slash commands return 409.
// @Tags agents
// @Accept json
// @Produce json
// @Param sessionId path string true "session id"
// @Param body body session.IntentRequest true "intent"
// @Success 200 {object} session.View
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 409 {object} APIErrorResponse
// @Router /api/v1/agents/sessions/{sessionId}/actions [post]
func (h *Handler) agentSessionIntent(c *gin.Context) {
	var req session.IntentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", err.Error()))
		return
	}
	if req.ActorID == "" {
		req.ActorID = currentActor(c)
	}
	view, err := h.sessionFor(c).Intent(c.Param("sessionId"), req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, errorBody("SESSION_NOT_FOUND", err.Error()))
			return
		}
		if errors.Is(err, session.ErrIntentRejected) {
			c.JSON(http.StatusConflict, errorBody("SESSION_INTENT_REJECTED", err.Error()))
			return
		}
		c.JSON(http.StatusBadRequest, errorBody("SESSION_INTENT_FAILED", err.Error()))
		return
	}
	if !h.requireRequestSpace(c, view.SpaceID) {
		return
	}
	_ = h.dbFor(c).Create(auditRow(view.SpaceID, currentActor(c), "agent.session_intent", map[string]any{
		"sessionId": view.ID, "action": req.Action, "runId": view.RunID,
	})).Error
	c.JSON(http.StatusOK, view)
}

// ListAgentSessionEvents godoc
// @Summary List session events (bound run events or synthesized turns)
// @Description When the session has a runId, returns run ledger events. Blank sessions project turns as session.turn envelopes.
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

// StreamAgentSession godoc
// @Summary Agent session live SSE
// @Description Bound sessions stream the run ledger; blank sessions poll ListEvents (assistant.* / session.turn).
// @Tags agents
// @Produce text/event-stream
// @Param sessionId path string true "session id"
// @Param afterSeq query int false "resume after this seq"
// @Param Last-Event-ID header string false "last received event id"
// @Success 200 {string} string "SSE stream"
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/agents/sessions/{sessionId}/stream [get]
func (h *Handler) streamAgentSession(c *gin.Context) {
	sessionID := c.Param("sessionId")
	view, err := h.sessionFor(c).Get(sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, errorBody("SESSION_NOT_FOUND", err.Error()))
		return
	}
	if !h.requireRequestSpace(c, view.SpaceID) {
		return
	}

	if runID := strings.TrimSpace(view.RunID); runID != "" {
		lastSeq := h.parseSSEResumeSeq(c, runID)
		h.streamRunLedgerSSE(c, runID, lastSeq)
		return
	}

	h.streamBlankSessionSSE(c, sessionID)
}

func (h *Handler) streamBlankSessionSSE(c *gin.Context, sessionID string) {
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, errorBody("SSE_UNSUPPORTED", "streaming not supported"))
		return
	}

	lastSeq := int64(0)
	if v := strings.TrimSpace(c.Query("afterSeq")); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n >= 0 {
			lastSeq = n
		}
	}
	lastID := strings.TrimSpace(c.GetHeader("Last-Event-ID"))
	if lastID == "" {
		lastID = strings.TrimSpace(c.Query("Last-Event-ID"))
	}
	if lastID == "" {
		lastID = strings.TrimSpace(c.Query("lastEventId"))
	}
	if lastID != "" {
		if n, err := strconv.ParseInt(lastID, 10, 64); err == nil && n > lastSeq {
			lastSeq = n
		} else if resp, err := h.sessionFor(c).ListEvents(sessionID, 0, 200); err == nil {
			for _, ev := range resp.Items {
				if ev.ID == lastID && ev.Seq > lastSeq {
					lastSeq = ev.Seq
				}
			}
		}
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
			if ev.Seq > lastSeq {
				lastSeq = ev.Seq
			}
		}
	}

	_, _ = c.Writer.Write([]byte(": session stream open\n\n"))
	flusher.Flush()

	svc := h.sessionFor(c)
	if resp, err := svc.ListEvents(sessionID, lastSeq, 100); err == nil {
		writeEvents(resp.Items)
	}

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-ticker.C:
			resp, err := svc.ListEvents(sessionID, lastSeq, 100)
			if err != nil {
				return
			}
			if len(resp.Items) == 0 {
				continue
			}
			writeEvents(resp.Items)
		}
	}
}

// ListAgentCommands godoc
// @Summary List agent slash/skill/MCP commands
// @Description Builtin /help /clear plus best-effort skills and active MCP tools for the space.
// @Tags agents
// @Produce json
// @Param spaceId query string false "space id (default: current space)"
// @Param repoRoot query string false "skills scan root" default(.)
// @Success 200 {object} session.CommandsResponse
// @Failure 403 {object} APIErrorResponse
// @Router /api/v1/agents/commands [get]
func (h *Handler) listAgentCommands(c *gin.Context) {
	spaceID := firstNonEmptyAPI(strings.TrimSpace(c.Query("spaceId")), currentSpace(c))
	if !h.requireRequestSpace(c, spaceID) {
		return
	}
	repoRoot := c.DefaultQuery("repoRoot", ".")
	c.JSON(http.StatusOK, session.ListCommandsForSpace(h.db.BindContext(c.Request.Context()), spaceID, repoRoot))
}

// ListAgentModels godoc
// @Summary List agent provider model seats
// @Description Builtin provider kinds: static, acp_sdk, execgo.
// @Tags agents
// @Produce json
// @Param spaceId query string false "space id (default: current space)"
// @Success 200 {object} session.ModelsResponse
// @Failure 403 {object} APIErrorResponse
// @Router /api/v1/agents/models [get]
func (h *Handler) listAgentModels(c *gin.Context) {
	spaceID := firstNonEmptyAPI(strings.TrimSpace(c.Query("spaceId")), currentSpace(c))
	if !h.requireRequestSpace(c, spaceID) {
		return
	}
	c.JSON(http.StatusOK, session.ListModels())
}

func queryTruthy(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
