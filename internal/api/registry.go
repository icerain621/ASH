package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ash-repwiki/ash/internal/registry"
)

// ListAgentAssets godoc
// @Summary List agent registry assets
// @Tags registry
// @Produce json
// @Param status query string false "draft|active|disabled"
// @Param limit query int false "max items"
// @Success 200 {object} registry.AgentAssetListResponse
// @Router /api/v1/agents/assets [get]
func (h *Handler) listAgentAssets(c *gin.Context) {
	space := currentSpace(c)
	if !h.requirePermission(c, permAgentsManage, space) {
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	resp, err := h.registryFor(c).ListAgentAssets(space, c.Query("status"), limit)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("REGISTRY_LIST_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, resp)
}

// CreateAgentAsset godoc
// @Summary Create agent registry asset
// @Tags registry
// @Accept json
// @Produce json
// @Param body body registry.CreateAgentAssetRequest true "asset"
// @Success 201 {object} registry.AgentAssetView
// @Router /api/v1/agents/assets [post]
func (h *Handler) createAgentAsset(c *gin.Context) {
	space := currentSpace(c)
	if !h.requirePermission(c, permAgentsManage, space) {
		return
	}
	var req registry.CreateAgentAssetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", err.Error()))
		return
	}
	if req.SpaceID == "" {
		req.SpaceID = space
	}
	view, err := h.registryFor(c).CreateAgentAsset(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("REGISTRY_CREATE_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusCreated, view)
}

// PatchAgentAssetStatus godoc
// @Summary Enable/disable agent asset
// @Tags registry
// @Accept json
// @Produce json
// @Param id path string true "asset id"
// @Param body body registry.PatchStatusRequest true "status"
// @Success 200 {object} registry.AgentAssetView
// @Router /api/v1/agents/assets/{id} [patch]
func (h *Handler) patchAgentAssetStatus(c *gin.Context) {
	space := currentSpace(c)
	if !h.requirePermission(c, permAgentsManage, space) {
		return
	}
	var req registry.PatchStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", err.Error()))
		return
	}
	view, err := h.registryFor(c).PatchAgentAssetStatus(c.Param("id"), req.Status)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("REGISTRY_PATCH_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, view)
}

// ListMemoryAssets godoc
// @Summary List memory registry assets
// @Tags registry
// @Produce json
// @Param status query string false "draft|active|disabled"
// @Param limit query int false "max items"
// @Success 200 {object} registry.MemoryAssetListResponse
// @Router /api/v1/memory/assets [get]
func (h *Handler) listMemoryAssets(c *gin.Context) {
	space := currentSpace(c)
	if !h.requirePermission(c, permMemoryManage, space) {
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	resp, err := h.registryFor(c).ListMemoryAssets(space, c.Query("status"), limit)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("REGISTRY_LIST_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, resp)
}

// CreateMemoryAsset godoc
// @Summary Create memory registry asset
// @Tags registry
// @Accept json
// @Produce json
// @Param body body registry.CreateMemoryAssetRequest true "asset"
// @Success 201 {object} registry.MemoryAssetView
// @Router /api/v1/memory/assets [post]
func (h *Handler) createMemoryAsset(c *gin.Context) {
	space := currentSpace(c)
	if !h.requirePermission(c, permMemoryManage, space) {
		return
	}
	var req registry.CreateMemoryAssetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", err.Error()))
		return
	}
	if req.SpaceID == "" {
		req.SpaceID = space
	}
	view, err := h.registryFor(c).CreateMemoryAsset(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("REGISTRY_CREATE_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusCreated, view)
}

// PatchMemoryAssetStatus godoc
// @Summary Enable/disable memory asset
// @Tags registry
// @Accept json
// @Produce json
// @Param id path string true "asset id"
// @Param body body registry.PatchStatusRequest true "status"
// @Success 200 {object} registry.MemoryAssetView
// @Router /api/v1/memory/assets/{id} [patch]
func (h *Handler) patchMemoryAssetStatus(c *gin.Context) {
	space := currentSpace(c)
	if !h.requirePermission(c, permMemoryManage, space) {
		return
	}
	var req registry.PatchStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", err.Error()))
		return
	}
	view, err := h.registryFor(c).PatchMemoryAssetStatus(c.Param("id"), req.Status)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("REGISTRY_PATCH_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, view)
}

func (h *Handler) registryFor(c *gin.Context) *registry.Service {
	return h.registry.WithDB(h.db.BindContext(c.Request.Context()))
}
