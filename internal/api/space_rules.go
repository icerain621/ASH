package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ash-repwiki/ash/internal/goal"
	"github.com/ash-repwiki/ash/internal/spacerules"
)

func (h *Handler) resolveRulesSpaceID(c *gin.Context) (string, bool) {
	spaceID := strings.TrimSpace(c.Param("spaceId"))
	if spaceID == "" {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", "spaceId required"))
		return "", false
	}
	if spaceID == "local" {
		if !h.requireRequestSpace(c, "local") {
			return "", false
		}
		return "local", true
	}
	space, ok := h.spaceForParam(c)
	if !ok {
		return "", false
	}
	return space.ID, true
}

// GetSpaceRules godoc
// @Summary Get Space Rules (builtin defaults if unset)
// @Tags spaces
// @Produce json
// @Param spaceId path string true "space id"
// @Success 200 {object} spacerules.View
// @Router /api/v1/spaces/{spaceId}/rules [get]
func (h *Handler) getSpaceRules(c *gin.Context) {
	spaceID, ok := h.resolveRulesSpaceID(c)
	if !ok {
		return
	}
	view, err := h.spaceRules.Get(spaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("SPACE_RULES_GET_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, view)
}

// PutSpaceRules godoc
// @Summary Upsert Space Rules in DB
// @Tags spaces
// @Accept json
// @Produce json
// @Param spaceId path string true "space id"
// @Param body body spacerules.PutRequest true "rules"
// @Success 200 {object} spacerules.View
// @Router /api/v1/spaces/{spaceId}/rules [put]
func (h *Handler) putSpaceRules(c *gin.Context) {
	spaceID, ok := h.resolveRulesSpaceID(c)
	if !ok {
		return
	}
	if spaceID != "local" && !h.requirePermission(c, permSpaceWrite, spaceID) {
		return
	}
	var req spacerules.PutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", err.Error()))
		return
	}
	if req.UpdatedBy == "" {
		req.UpdatedBy = currentActor(c)
	}
	view, err := h.spaceRules.Put(spaceID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("SPACE_RULES_PUT_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, view)
}

// ImportSpaceRules godoc
// @Summary Import Space Rules from .ash/rules.yaml
// @Tags spaces
// @Accept json
// @Produce json
// @Param spaceId path string true "space id"
// @Param body body spacerules.SyncRequest true "sync"
// @Success 200 {object} spacerules.View
// @Router /api/v1/spaces/{spaceId}/rules/import [post]
func (h *Handler) importSpaceRules(c *gin.Context) {
	spaceID, ok := h.resolveRulesSpaceID(c)
	if !ok {
		return
	}
	if spaceID != "local" && !h.requirePermission(c, permSpaceWrite, spaceID) {
		return
	}
	var req spacerules.SyncRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", err.Error()))
		return
	}
	if req.UpdatedBy == "" {
		req.UpdatedBy = currentActor(c)
	}
	view, err := h.spaceRules.ImportFromFile(spaceID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("SPACE_RULES_IMPORT_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, view)
}

// ExportSpaceRules godoc
// @Summary Export Space Rules to .ash/rules.yaml
// @Tags spaces
// @Accept json
// @Produce json
// @Param spaceId path string true "space id"
// @Param body body spacerules.SyncRequest true "sync"
// @Success 200 {object} spacerules.View
// @Router /api/v1/spaces/{spaceId}/rules/export [post]
func (h *Handler) exportSpaceRules(c *gin.Context) {
	spaceID, ok := h.resolveRulesSpaceID(c)
	if !ok {
		return
	}
	if spaceID != "local" && !h.requirePermission(c, permSpaceWrite, spaceID) {
		return
	}
	var req spacerules.SyncRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", err.Error()))
		return
	}
	view, err := h.spaceRules.ExportToFile(spaceID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("SPACE_RULES_EXPORT_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, view)
}

// PreviewSpaceRules godoc
// @Summary Preview Goal routing under Space Rules
// @Tags spaces
// @Accept json
// @Produce json
// @Param spaceId path string true "space id"
// @Param body body spacerules.PreviewRequest true "preview"
// @Success 200 {object} spacerules.PreviewResponse
// @Router /api/v1/spaces/{spaceId}/rules/preview [post]
func (h *Handler) previewSpaceRules(c *gin.Context) {
	spaceID, ok := h.resolveRulesSpaceID(c)
	if !ok {
		return
	}
	var req spacerules.PreviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", err.Error()))
		return
	}
	view, err := h.spaceRules.Get(spaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("SPACE_RULES_GET_FAILED", err.Error()))
		return
	}
	routed, err := goal.RouteWithDoc(req.Goal, h.scenarios, req.RepoRoot, view.Document)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("SPACE_RULES_PREVIEW_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, spacerules.PreviewResponse{
		ScenarioName: routed.ScenarioName, RouteReason: routed.Reason,
		PolicyProfile: routed.PolicyProfile, Inputs: routed.Inputs,
	})
}
