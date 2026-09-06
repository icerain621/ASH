package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ash-repwiki/ash/internal/skills"
)

type skillCatalogInstallRequest struct {
	RepoRoot string `json:"repoRoot"`
	SpaceID  string `json:"spaceId"`
	Name     string `json:"name" binding:"required"`
	Version  string `json:"version"`
}

// ListSkillCatalog godoc
// @Summary List org skill catalog entries (filesystem/HTTP; no public marketplace)
// @Tags skills
// @Produce json
// @Param repoRoot query string false "repository root for .ash/skill-catalog.json" default(.)
// @Success 200 {object} skills.CatalogListResponse
// @Failure 400 {object} skills.CatalogListResponse
// @Router /api/v1/skills/catalog [get]
func (h *Handler) listSkillCatalog(c *gin.Context) {
	out, err := skills.ListCatalog(c.DefaultQuery("repoRoot", "."))
	if out == nil {
		out = &skills.CatalogListResponse{OK: false, Message: err.Error(), Items: []skills.CatalogItem{}}
	}
	if err != nil || !out.OK {
		c.JSON(http.StatusBadRequest, out)
		return
	}
	c.JSON(http.StatusOK, out)
}

// InstallSkillFromCatalog godoc
// @Summary Install a skill pack referenced by the org catalog
// @Tags skills
// @Accept json
// @Produce json
// @Param body body skillCatalogInstallRequest true "catalog name (+ optional version)"
// @Success 200 {object} skills.PackInstallResult
// @Failure 400 {object} map[string]interface{}
// @Router /api/v1/skills/catalog/install [post]
func (h *Handler) installSkillFromCatalog(c *gin.Context) {
	var req skillCatalogInstallRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorBody("SKILL_CATALOG_BAD_REQUEST", err.Error()))
		return
	}
	repo := strings.TrimSpace(req.RepoRoot)
	if repo == "" {
		repo = c.DefaultQuery("repoRoot", ".")
	}
	inst, err := skills.InstallFromCatalog(repo, req.SpaceID, req.Name, req.Version)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("SKILL_CATALOG_INSTALL_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, inst)
}
