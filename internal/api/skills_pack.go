package api

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ash-repwiki/ash/internal/skills"
)

type skillPackRequest struct {
	RepoRoot   string `json:"repoRoot"`
	SpaceID    string `json:"spaceId"`
	PackPath   string `json:"packPath"`
	PackBase64 string `json:"packBase64"`
	Signature  string `json:"signature"`
}

// VerifySkillPack godoc
// @Summary Verify a signed private skill pack (dry-run)
// @Tags skills
// @Accept json
// @Produce json
// @Param body body skillPackRequest true "pack path or base64 + signature"
// @Success 200 {object} skills.PackVerifyResult
// @Failure 400 {object} skills.PackVerifyResult
// @Router /api/v1/skills/packs/verify [post]
func (h *Handler) verifySkillPack(c *gin.Context) {
	var req skillPackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorBody("SKILL_PACK_BAD_REQUEST", err.Error()))
		return
	}
	zipBytes, err := loadSkillPackBytes(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("SKILL_PACK_LOAD_FAILED", err.Error()))
		return
	}
	vr, _, _, err := skills.VerifyPackBytes(zipBytes, req.Signature)
	if vr == nil {
		vr = &skills.PackVerifyResult{OK: false, Message: err.Error()}
	}
	if err != nil || !vr.OK {
		c.JSON(http.StatusBadRequest, vr)
		return
	}
	c.JSON(http.StatusOK, vr)
}

// InstallSkillPack godoc
// @Summary Install a signed private skill pack under .ash/skills
// @Tags skills
// @Accept json
// @Produce json
// @Param body body skillPackRequest true "pack path or base64 + signature"
// @Success 200 {object} skills.PackInstallResult
// @Failure 400 {object} map[string]interface{}
// @Router /api/v1/skills/packs/install [post]
func (h *Handler) installSkillPack(c *gin.Context) {
	var req skillPackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorBody("SKILL_PACK_BAD_REQUEST", err.Error()))
		return
	}
	zipBytes, err := loadSkillPackBytes(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("SKILL_PACK_LOAD_FAILED", err.Error()))
		return
	}
	repo := strings.TrimSpace(req.RepoRoot)
	if repo == "" {
		repo = c.DefaultQuery("repoRoot", ".")
	}
	inst, err := skills.InstallPackBytes(repo, req.SpaceID, zipBytes, req.Signature)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("SKILL_PACK_INSTALL_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, inst)
}

func loadSkillPackBytes(req skillPackRequest) ([]byte, error) {
	if b64 := strings.TrimSpace(req.PackBase64); b64 != "" {
		return base64.StdEncoding.DecodeString(b64)
	}
	path := strings.TrimSpace(req.PackPath)
	if path == "" {
		return nil, fmt.Errorf("packPath or packBase64 required")
	}
	return os.ReadFile(filepath.Clean(path))
}
