package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ash-repwiki/ash/internal/evolve"
)

type reviewsQueueResponse struct {
	Items []evolve.Item `json:"items"`
	Queue string        `json:"queue,omitempty"`
}

// ListReviewsQueue godoc
// @Summary List unified review queue (memory + orchestration)
// @Tags reviews
// @Produce json
// @Param queue query string false "memory|orchestration|all"
// @Param limit query int false "max items" default(50)
// @Success 200 {object} reviewsQueueResponse
// @Router /api/v1/reviews/queue [get]
func (h *Handler) listReviewsQueue(c *gin.Context) {
	space := currentSpace(c)
	// Dev/jwt: feedback:read or memory:read is enough to view the unified queue.
	if !h.requirePermission(c, permFeedbackRead, space) {
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	resp, err := h.evolveFor(c).ListQueue(space, c.Query("queue"), limit)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("REVIEWS_QUEUE_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, reviewsQueueResponse{Items: resp.Items, Queue: resp.Queue})
}

// DecideReview godoc
// @Summary Approve or reject a review queue item
// @Tags reviews
// @Accept json
// @Produce json
// @Param reviewId path string true "memory:<id> or harness_profile:<id>"
// @Param body body evolve.DecideRequest true "decision"
// @Success 200 {object} evolve.DecideResponse
// @Failure 400 {object} APIErrorResponse
// @Router /api/v1/reviews/{reviewId}/decide [post]
func (h *Handler) decideReview(c *gin.Context) {
	space := currentSpace(c)
	var req evolve.DecideRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", err.Error()))
		return
	}
	if req.ActorID == "" {
		req.ActorID = currentActor(c)
	}
	reviewID := c.Param("reviewId")
	tt, _, ok := evolve.ParseItemID(reviewID)
	if !ok {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REVIEW_ID", "expected memory:<id> or harness_profile:<id>"))
		return
	}
	switch tt {
	case "memory":
		if !h.requirePermission(c, permMemoryReview, space) {
			return
		}
	case "harness_profile":
		if !h.requirePermission(c, permFeedbackWrite, space) {
			return
		}
	case "scenario_patch":
		if !h.requirePermission(c, permFeedbackWrite, space) {
			return
		}
	default:
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REVIEW_ID", "unsupported review target"))
		return
	}
	resp, err := h.evolveFor(c).Decide(space, reviewID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("REVIEW_DECIDE_FAILED", err.Error()))
		return
	}
	_ = h.dbFor(c).Create(auditRow(space, currentActor(c), "review.decided", map[string]any{
		"reviewId": reviewID, "decision": req.Decision, "reason": req.Reason,
		"targetType": resp.TargetType, "targetId": resp.TargetID,
	}))
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) evolveFor(c *gin.Context) *evolve.Service {
	db := h.db.BindContext(c.Request.Context())
	mem := h.memory.WithContext(c.Request.Context())
	har := h.harness.WithContext(c.Request.Context())
	patches := h.patches.WithContext(c.Request.Context())
	return h.evolve.WithContext(db, mem, har, patches)
}
