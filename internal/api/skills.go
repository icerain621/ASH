package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ash-repwiki/ash/internal/skills"
)

// ListSkills godoc
// @Summary List SKILL.md skills in a repository
// @Tags skills
// @Produce json
// @Param repoRoot query string false "repository root" default(.)
// @Success 200 {object} skills.ListResponse
// @Router /api/v1/skills [get]
func (h *Handler) listSkills(c *gin.Context) {
	out, err := skills.ScanRepo(c.DefaultQuery("repoRoot", "."))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("SKILLS_LIST_FAILED", err.Error()))
		return
	}
	// omit bodies in list
	for i := range out.Items {
		out.Items[i].Body = ""
	}
	c.JSON(http.StatusOK, out)
}

// GetSkill godoc
// @Summary Get a skill by id (includes body)
// @Tags skills
// @Produce json
// @Param skillId path string true "skill id"
// @Param repoRoot query string false "repository root" default(.)
// @Success 200 {object} skills.Skill
// @Router /api/v1/skills/{skillId} [get]
func (h *Handler) getSkill(c *gin.Context) {
	view, err := skills.Get(c.DefaultQuery("repoRoot", "."), c.Param("skillId"))
	if err != nil {
		code := http.StatusNotFound
		if strings.Contains(err.Error(), "required") {
			code = http.StatusBadRequest
		}
		c.JSON(code, errorBody("SKILLS_GET_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, view)
}
