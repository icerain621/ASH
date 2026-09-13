package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ash-repwiki/ash/internal/spacepolicy"
)

type spacePolicyResponse struct {
	Pack      *spacepolicy.Pack            `json:"pack"`
	Effective *spacepolicy.EffectivePolicy `json:"effective"`
}

// GetSpacePolicy godoc
// @Summary Get space policy pack and effective merge
// @Tags spaces
// @Produce json
// @Param spaceId path string true "space id"
// @Success 200 {object} spacePolicyResponse
// @Router /api/v1/spaces/{spaceId}/policy [get]
func (h *Handler) getSpacePolicy(c *gin.Context) {
	spaceID := c.Param("spaceId")
	if spaceID != "local" && !h.requirePermission(c, permSpacesPolicy, spaceID) {
		return
	}
	pack, err := h.spacePolicyFor(c).GetPack(spaceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("SPACE_POLICY_FAILED", err.Error()))
		return
	}
	eff, err := h.spacePolicyFor(c).EffectivePolicy(spaceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("SPACE_POLICY_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, spacePolicyResponse{Pack: pack, Effective: eff})
}

// PutSpacePolicy godoc
// @Summary Upsert space policy pack
// @Tags spaces
// @Accept json
// @Produce json
// @Param spaceId path string true "space id"
// @Param body body spacepolicy.PutPackRequest true "pack"
// @Success 200 {object} spacePolicyResponse
// @Router /api/v1/spaces/{spaceId}/policy [put]
func (h *Handler) putSpacePolicy(c *gin.Context) {
	spaceID := c.Param("spaceId")
	if spaceID != "local" && !h.requirePermission(c, permSpacesPolicy, spaceID) {
		return
	}
	var req spacepolicy.PutPackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", err.Error()))
		return
	}
	pack, err := h.spacePolicyFor(c).PutPack(spaceID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("SPACE_POLICY_FAILED", err.Error()))
		return
	}
	eff, err := h.spacePolicyFor(c).EffectivePolicy(spaceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("SPACE_POLICY_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, spacePolicyResponse{Pack: pack, Effective: eff})
}

func (h *Handler) spacePolicyFor(c *gin.Context) *spacepolicy.Service {
	return h.spacePolicy.WithDB(h.db.BindContext(c.Request.Context()))
}
