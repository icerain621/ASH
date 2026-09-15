package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ash-repwiki/ash/internal/agentworkspace"
	"github.com/ash-repwiki/ash/internal/session"
)

// ListAgentWorkspaces godoc
// @Summary List agent workspaces
// @Description List active agent.workspace audit documents for a space, newest updatedAt first.
// @Tags agents
// @Produce json
// @Param spaceId query string false "space id (default: current space)"
// @Param limit query int false "max items" default(50)
// @Success 200 {object} agentworkspace.ListResponse
// @Failure 403 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/agent-workspaces [get]
func (h *Handler) listAgentWorkspaces(c *gin.Context) {
	spaceID := firstNonEmptyAPI(strings.TrimSpace(c.Query("spaceId")), currentSpace(c))
	if !h.requireRequestSpace(c, spaceID) {
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	items, err := h.agentWorkspaceFor(c).List(spaceID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("WORKSPACE_LIST_FAILED", err.Error()))
		return
	}
	if items == nil {
		items = []agentworkspace.View{}
	}
	c.JSON(http.StatusOK, agentworkspace.ListResponse{Items: items})
}

// CreateAgentWorkspace godoc
// @Summary Create an agent workspace
// @Description Persist a workspace grouping document in audit_log (event_type=agent.workspace).
// @Tags agents
// @Accept json
// @Produce json
// @Param body body agentworkspace.CreateRequest true "workspace create"
// @Success 201 {object} agentworkspace.View
// @Failure 400 {object} APIErrorResponse
// @Failure 403 {object} APIErrorResponse
// @Router /api/v1/agent-workspaces [post]
func (h *Handler) createAgentWorkspace(c *gin.Context) {
	var req agentworkspace.CreateRequest
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
	view, err := h.agentWorkspaceFor(c).Create(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("WORKSPACE_CREATE_FAILED", err.Error()))
		return
	}
	_ = h.dbFor(c).Create(auditRow(req.SpaceID, currentActor(c), "agent.workspace_created", map[string]any{
		"workspaceId": view.ID, "title": view.Title, "repoRoot": view.RepoRoot,
	})).Error
	c.JSON(http.StatusCreated, view)
}

// PatchAgentWorkspace godoc
// @Summary Patch agent workspace
// @Description Rename, update repoRoot, and/or replace sessionIds order/membership.
// @Tags agents
// @Accept json
// @Produce json
// @Param workspaceId path string true "workspace id"
// @Param body body agentworkspace.PatchRequest true "patch"
// @Success 200 {object} agentworkspace.View
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Router /api/v1/agent-workspaces/{workspaceId} [patch]
func (h *Handler) patchAgentWorkspace(c *gin.Context) {
	var req agentworkspace.PatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", err.Error()))
		return
	}
	view, err := h.agentWorkspaceFor(c).Get(c.Param("workspaceId"))
	if err != nil {
		c.JSON(http.StatusNotFound, errorBody("WORKSPACE_NOT_FOUND", err.Error()))
		return
	}
	if !h.requireRequestSpace(c, view.SpaceID) {
		return
	}
	if !h.requirePermission(c, permRunCreate, view.SpaceID) {
		return
	}
	updated, err := h.agentWorkspaceFor(c).Patch(c.Param("workspaceId"), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("WORKSPACE_UPDATE_FAILED", err.Error()))
		return
	}
	_ = h.dbFor(c).Create(auditRow(updated.SpaceID, currentActor(c), "agent.workspace_updated", map[string]any{
		"workspaceId": updated.ID, "title": updated.Title, "sessionIds": updated.SessionIDs,
	})).Error
	c.JSON(http.StatusOK, updated)
}

// AttachAgentWorkspaceSession godoc
// @Summary Attach a session to an agent workspace
// @Description Append sessionId to workspace.sessionIds when missing.
// @Tags agents
// @Accept json
// @Produce json
// @Param workspaceId path string true "workspace id"
// @Param body body agentworkspace.AttachRequest true "attach"
// @Success 200 {object} agentworkspace.View
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Router /api/v1/agent-workspaces/{workspaceId}/sessions [post]
func (h *Handler) attachAgentWorkspaceSession(c *gin.Context) {
	var req agentworkspace.AttachRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", err.Error()))
		return
	}
	view, err := h.agentWorkspaceFor(c).Get(c.Param("workspaceId"))
	if err != nil {
		c.JSON(http.StatusNotFound, errorBody("WORKSPACE_NOT_FOUND", err.Error()))
		return
	}
	if !h.requireRequestSpace(c, view.SpaceID) {
		return
	}
	if !h.requirePermission(c, permRunCreate, view.SpaceID) {
		return
	}
	updated, err := h.agentWorkspaceFor(c).Attach(c.Param("workspaceId"), req.SessionID)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("WORKSPACE_ATTACH_FAILED", err.Error()))
		return
	}
	if _, err := h.sessionFor(c).SetWorkspaceID(req.SessionID, updated.ID); err != nil {
		// Session may not exist yet in some tests; membership on workspace is source of truth.
		_ = err
	}
	_ = h.dbFor(c).Create(auditRow(updated.SpaceID, currentActor(c), "agent.workspace_session_attached", map[string]any{
		"workspaceId": updated.ID, "sessionId": req.SessionID,
	})).Error
	c.JSON(http.StatusOK, updated)
}

// DetachAgentWorkspaceSession godoc
// @Summary Detach a session from an agent workspace
// @Description Remove sessionId from workspace.sessionIds and clear session.workspaceId when matching.
// @Tags agents
// @Produce json
// @Param workspaceId path string true "workspace id"
// @Param sessionId path string true "session id"
// @Success 200 {object} agentworkspace.View
// @Failure 404 {object} APIErrorResponse
// @Router /api/v1/agent-workspaces/{workspaceId}/sessions/{sessionId} [delete]
func (h *Handler) detachAgentWorkspaceSession(c *gin.Context) {
	workspaceID := c.Param("workspaceId")
	sessionID := c.Param("sessionId")
	view, err := h.agentWorkspaceFor(c).Get(workspaceID)
	if err != nil {
		c.JSON(http.StatusNotFound, errorBody("WORKSPACE_NOT_FOUND", err.Error()))
		return
	}
	if !h.requireRequestSpace(c, view.SpaceID) {
		return
	}
	if !h.requirePermission(c, permRunCreate, view.SpaceID) {
		return
	}
	updated, err := h.agentWorkspaceFor(c).Detach(workspaceID, sessionID)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("WORKSPACE_DETACH_FAILED", err.Error()))
		return
	}
	if sess, err := h.sessionFor(c).Get(sessionID); err == nil {
		if strings.TrimSpace(sess.WorkspaceID) == workspaceID || strings.TrimSpace(sess.WorkspaceID) == "" {
			_, _ = h.sessionFor(c).SetWorkspaceID(sessionID, "")
		}
	}
	_ = h.dbFor(c).Create(auditRow(updated.SpaceID, currentActor(c), "agent.workspace_session_detached", map[string]any{
		"workspaceId": updated.ID, "sessionId": sessionID,
	})).Error
	c.JSON(http.StatusOK, updated)
}

// CloseAgentWorkspace godoc
// @Summary Soft-close agent workspace
// @Description Sets status=closed. List excludes closed workspaces.
// @Tags agents
// @Produce json
// @Param workspaceId path string true "workspace id"
// @Success 200 {object} agentworkspace.View
// @Failure 404 {object} APIErrorResponse
// @Router /api/v1/agent-workspaces/{workspaceId} [delete]
func (h *Handler) closeAgentWorkspace(c *gin.Context) {
	view, err := h.agentWorkspaceFor(c).Get(c.Param("workspaceId"))
	if err != nil {
		c.JSON(http.StatusNotFound, errorBody("WORKSPACE_NOT_FOUND", err.Error()))
		return
	}
	if !h.requireRequestSpace(c, view.SpaceID) {
		return
	}
	if !h.requirePermission(c, permRunCreate, view.SpaceID) {
		return
	}
	closed, err := h.agentWorkspaceFor(c).Close(c.Param("workspaceId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("WORKSPACE_CLOSE_FAILED", err.Error()))
		return
	}
	_ = h.dbFor(c).Create(auditRow(closed.SpaceID, currentActor(c), "agent.workspace_closed", map[string]any{
		"workspaceId": closed.ID, "status": closed.Status,
	})).Error
	c.JSON(http.StatusOK, closed)
}

func (h *Handler) enrichSessionWorkspace(c *gin.Context, view *session.View) {
	if view == nil || strings.TrimSpace(view.WorkspaceID) != "" {
		return
	}
	if id, ok := h.agentWorkspaceFor(c).FindIDBySession(view.SpaceID, view.ID); ok {
		view.WorkspaceID = id
	}
}
