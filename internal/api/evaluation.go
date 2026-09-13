package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ash-repwiki/ash/internal/scoring"
)

// getSpaceEvaluation godoc
// @Summary Space dual-core evaluation
// @Description Aggregate Quality/Safety/Efficiency/Governance plus seal/replay/memory-link health and scenario ranking (GV06–07).
// @Tags spaces
// @Produce json
// @Param spaceId path string true "space id"
// @Param from query string false "RFC3339 start time"
// @Param to query string false "RFC3339 end time"
// @Success 200 {object} scoring.Evaluation
// @Failure 400 {object} APIErrorResponse
// @Failure 403 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/spaces/{spaceId}/evaluation [get]
func (h *Handler) getSpaceEvaluation(c *gin.Context) {
	spaceID := strings.TrimSpace(c.Param("spaceId"))
	if spaceID == "" {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", "spaceId is required"))
		return
	}
	if !h.requireRequestSpace(c, spaceID) {
		return
	}
	if !h.requirePermission(c, permObservabilityRead, spaceID) {
		return
	}
	from, err := parseOptionalTime(c.Query("from"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_FROM", err.Error()))
		return
	}
	to, err := parseOptionalTime(c.Query("to"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_TO", err.Error()))
		return
	}
	svc := h.scoringFor(c)
	if svc == nil {
		c.JSON(http.StatusInternalServerError, errorBody("SPACE_EVALUATION_FAILED", "scoring service unavailable"))
		return
	}
	out, err := svc.Evaluate(scoring.EvaluationRequest{SpaceID: spaceID, From: from, To: to})
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("SPACE_EVALUATION_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, out)
}
