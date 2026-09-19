package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ash-repwiki/ash/internal/spacepolicy"
)

// getSpaceQuotas godoc
// @Summary Space quota limits and usage
// @Description DX68: bodyJson.quotas limits plus active concurrent run count. Zero limit = unlimited. Token proxy usage is 0 until accounting ships.
// @Tags spaces
// @Produce json
// @Param spaceId path string true "space id"
// @Success 200 {object} spacepolicy.QuotaStatus
// @Failure 400 {object} APIErrorResponse
// @Failure 403 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/spaces/{spaceId}/quotas [get]
func (h *Handler) getSpaceQuotas(c *gin.Context) {
	spaceID := strings.TrimSpace(c.Param("spaceId"))
	if spaceID == "" {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", "spaceId is required"))
		return
	}
	if spaceID != "local" && !h.requirePermission(c, permSpacesPolicy, spaceID) {
		return
	}
	pack, err := h.spacePolicyFor(c).GetPack(spaceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("SPACE_POLICY_FAILED", err.Error()))
		return
	}
	bodyJSON := "{}"
	if pack != nil && pack.BodyJSON != "" {
		bodyJSON = pack.BodyJSON
	}
	active, err := h.runsFor(c).CountActiveRuns(spaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("SPACE_QUOTA_STATUS_FAILED", err.Error()))
		return
	}
	st, err := spacepolicy.BuildQuotaStatus(spaceID, bodyJSON, int(active))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("SPACE_QUOTA_STATUS_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, st)
}
