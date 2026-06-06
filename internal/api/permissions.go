package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ash-repwiki/ash/internal/authz"
)

// PermissionMatrix godoc
// @Summary M2 permission matrix for current space
// @Tags permissions
// @Produce json
// @Success 200 {object} authz.MatrixResponse
// @Failure 403 {object} APIErrorResponse
// @Router /api/v1/permissions/matrix [get]
func (h *Handler) permissionMatrix(c *gin.Context) {
	space := currentSpace(c)
	if !h.requirePermission(c, permRoleRead, space) {
		return
	}
	orgID, _ := h.spaceOrgID(c, space)
	resp, err := authz.BuildMatrix(h.db.BindContext(c.Request.Context()), space, orgID, currentRole(c), currentActor(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("PERMISSION_MATRIX_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, resp)
}

// SpacePermissionMatrix godoc
// @Summary M2 permission matrix for a specific space
// @Tags permissions
// @Produce json
// @Param spaceId path string true "space id"
// @Success 200 {object} authz.MatrixResponse
// @Failure 403 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Router /api/v1/spaces/{spaceId}/permissions/matrix [get]
func (h *Handler) spacePermissionMatrix(c *gin.Context) {
	space, ok := h.spaceForParam(c)
	if !ok {
		return
	}
	if !h.requirePermission(c, permRoleRead, space.ID) {
		return
	}
	resp, err := authz.BuildMatrix(h.db.BindContext(c.Request.Context()), space.ID, space.OrgID, currentRole(c), currentActor(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("PERMISSION_MATRIX_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, resp)
}
