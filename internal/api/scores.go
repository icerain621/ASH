package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ash-repwiki/ash/internal/evolve"
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

type createScoreAppealRequest struct {
	Reason  string `json:"reason" binding:"required"`
	ActorID string `json:"actorId,omitempty"`
}

// CreateScoreAppeal godoc
// @Summary Open a score appeal review queue item
// @Tags scores
// @Accept json
// @Produce json
// @Param scoreEventId path string true "score event id"
// @Param body body createScoreAppealRequest true "appeal reason"
// @Success 201 {object} evolve.Item
// @Failure 400 {object} APIErrorResponse
// @Router /api/v1/scores/{scoreEventId}/appeal [post]
func (h *Handler) createScoreAppeal(c *gin.Context) {
	space := currentSpace(c)
	if !h.requirePermission(c, permFeedbackWrite, space) {
		return
	}
	var req createScoreAppealRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", err.Error()))
		return
	}
	actor := req.ActorID
	if actor == "" {
		actor = currentActor(c)
	}
	var item *evolve.Item
	var err error
	item, err = h.evolveFor(c).CreateAppeal(space, c.Param("scoreEventId"), actor, req.Reason)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("SCORE_APPEAL_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusCreated, item)
}
