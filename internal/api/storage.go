package api

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"

	"github.com/ash-repwiki/ash/internal/artifactstore"
	"github.com/ash-repwiki/ash/internal/config"
	"github.com/ash-repwiki/ash/internal/store"
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
	dbProf, _ := store.DatabaseProfile(h.db.DataDir(), os.Getenv("ASH_DATABASE_URL"))
	if h.db.Dialect() == "postgres" && store.PostgresRLSEnabled() {
		dbProf.PostgresRLSPolicyCount, _ = store.CountPostgresRLSPolicies(h.db)
	}
	resp := StorageProfileResponse{
		Database: DatabaseProfile{
			Dialect:                h.db.Dialect(),
			URLConfigured:          cfg.DatabaseURL != "",
			DataDir:                h.db.DataDir(),
			PostgresRLSEnabled:     dbProf.PostgresRLSEnabled,
			PostgresRLSForce:       dbProf.PostgresRLSForce,
			PostgresRLSPolicyCount: dbProf.PostgresRLSPolicyCount,
		},
		ArtifactStore: artifactstore.Describe(cfg.ArtifactStore, h.db.DataDir()),
	}
	c.JSON(http.StatusOK, resp)
}
