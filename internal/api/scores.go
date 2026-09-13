package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ash-repwiki/ash/internal/scoring"
)

// GetScoreRubrics godoc
// @Summary Default review rubric schema
// @Tags scores
// @Produce json
// @Success 200 {object} scoring.RubricSchema
// @Router /api/v1/scores/rubrics [get]
func (h *Handler) getScoreRubrics(c *gin.Context) {
	space := currentSpace(c)
	if !h.requirePermission(c, permFeedbackRead, space) {
		return
	}
	c.JSON(http.StatusOK, scoring.DefaultRubricSchema())
}
