package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ash-repwiki/ash/internal/store"
)

// requireRequestSpace rejects when record space does not match the request context space.
func (h *Handler) requireRequestSpace(c *gin.Context, recordSpaceID string) bool {
	if err := store.EnforceSpaceAccess(recordSpaceID, currentSpace(c)); err != nil {
		c.AbortWithStatusJSON(http.StatusForbidden, errorBody("SPACE_ACCESS_DENIED", err.Error()))
		return false
	}
	return true
}

// requireTargetSpace ensures a write target space matches the authenticated request space.
func (h *Handler) requireTargetSpace(c *gin.Context, targetSpaceID string) bool {
	target := firstNonEmptyAPI(strings.TrimSpace(targetSpaceID), currentSpace(c))
	return h.requireRequestSpace(c, target)
}
