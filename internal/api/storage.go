package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ash-repwiki/ash/internal/artifactstore"
	"github.com/ash-repwiki/ash/internal/config"
)

// GetStorageProfile godoc
// @Summary Get storage profile
// @Tags storage
// @Produce json
// @Success 200 {object} StorageProfileResponse
// @Failure 403 {object} APIErrorResponse
// @Router /api/v1/storage/profile [get]
func (h *Handler) getStorageProfile(c *gin.Context) {
	if !h.requirePermission(c, permStorageRead, currentSpace(c)) {
		return
	}
	cfg := config.Load()
	resp := StorageProfileResponse{
		Database: DatabaseProfile{
			Dialect:       h.db.Dialect(),
			URLConfigured: cfg.DatabaseURL != "",
			DataDir:       h.db.DataDir(),
		},
		ArtifactStore: artifactstore.Describe(cfg.ArtifactStore, h.db.DataDir()),
	}
	c.JSON(http.StatusOK, resp)
}
