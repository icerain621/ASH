package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/ash-repwiki/ash/internal/goal"
)

type approveGoalPlanRequest struct {
	ActorID string `json:"actorId"`
	Reason  string `json:"reason"`
}

type rejectGoalPlanRequest struct {
	ActorID string `json:"actorId"`
	Reason  string `json:"reason"`
}

// FromGoal godoc
// @Summary Create a plan draft from a natural-language goal
// @Tags runs
// @Accept json
// @Produce json
// @Param body body goal.FromGoalRequest true "goal"
// @Success 201 {object} goal.PlanView
// @Failure 400 {object} APIErrorResponse
// @Router /api/v1/runs/from-goal [post]
func (h *Handler) fromGoal(c *gin.Context) {
	var req goal.FromGoalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", err.Error()))
		return
	}
	req.SpaceID = firstNonEmptyAPI(strings.TrimSpace(req.SpaceID), currentSpace(c))
	if !h.requireTargetSpace(c, req.SpaceID) {
		return
	}
	if !h.requirePermission(c, permRunCreate, req.SpaceID) {
		return
	}
	if req.CreatedBy == "" {
		req.CreatedBy = currentActor(c)
	}
	if req.ActorRole == "" {
		req.ActorRole = currentRole(c)
	}
	view, err := h.goalFor(c).FromGoal(req)
	if err != nil && view == nil {
		c.JSON(http.StatusBadRequest, errorBody("GOAL_ROUTE_FAILED", err.Error()))
		return
	}
	status := http.StatusCreated
	if req.AutoApprove && view != nil && view.RunID != "" {
		status = http.StatusCreated
	}
	c.JSON(status, view)
}

// GetGoalPlan godoc
// @Summary Get goal plan by id
// @Tags runs
// @Produce json
// @Param planId path string true "plan id"
// @Success 200 {object} goal.PlanView
// @Failure 404 {object} APIErrorResponse
// @Router /api/v1/runs/plans/{planId} [get]
func (h *Handler) getGoalPlan(c *gin.Context) {
	view, err := h.goalFor(c).Get(c.Param("planId"))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, errorBody("GOAL_PLAN_NOT_FOUND", "plan not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, errorBody("GOAL_PLAN_GET_FAILED", err.Error()))
		return
	}
	if !h.requireRequestSpace(c, view.SpaceID) {
		return
	}
	c.JSON(http.StatusOK, view)
}

// ApproveGoalPlan godoc
// @Summary Approve goal plan and start a run
// @Tags runs
// @Accept json
// @Produce json
// @Param planId path string true "plan id"
// @Param body body approveGoalPlanRequest false "approve"
// @Success 200 {object} goal.PlanView
// @Failure 400 {object} APIErrorResponse
// @Router /api/v1/runs/plans/{planId}/approve [post]
func (h *Handler) approveGoalPlan(c *gin.Context) {
	view, err := h.goalFor(c).Get(c.Param("planId"))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, errorBody("GOAL_PLAN_NOT_FOUND", "plan not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, errorBody("GOAL_PLAN_GET_FAILED", err.Error()))
		return
	}
	if !h.requireRequestSpace(c, view.SpaceID) {
		return
	}
	if !h.requirePermission(c, permRunCreate, view.SpaceID) {
		return
	}
	var req approveGoalPlanRequest
	_ = c.ShouldBindJSON(&req)
	actor := firstNonEmptyAPI(req.ActorID, currentActor(c))
	reason := firstNonEmptyAPI(req.Reason, "approved from API")
	out, err := h.goalFor(c).Approve(c.Param("planId"), actor, reason, currentRole(c))
	if err != nil && out == nil {
		c.JSON(http.StatusBadRequest, errorBody("GOAL_PLAN_APPROVE_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, out)
}

// RejectGoalPlan godoc
// @Summary Reject goal plan draft
// @Tags runs
// @Accept json
// @Produce json
// @Param planId path string true "plan id"
// @Param body body rejectGoalPlanRequest false "reject"
// @Success 200 {object} goal.PlanView
// @Failure 400 {object} APIErrorResponse
// @Router /api/v1/runs/plans/{planId}/reject [post]
func (h *Handler) rejectGoalPlan(c *gin.Context) {
	view, err := h.goalFor(c).Get(c.Param("planId"))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, errorBody("GOAL_PLAN_NOT_FOUND", "plan not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, errorBody("GOAL_PLAN_GET_FAILED", err.Error()))
		return
	}
	if !h.requireRequestSpace(c, view.SpaceID) {
		return
	}
	if !h.requirePermission(c, permRunCreate, view.SpaceID) {
		return
	}
	var req rejectGoalPlanRequest
	_ = c.ShouldBindJSON(&req)
	actor := firstNonEmptyAPI(req.ActorID, currentActor(c))
	reason := firstNonEmptyAPI(req.Reason, "rejected from API")
	out, err := h.goalFor(c).Reject(c.Param("planId"), actor, reason)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("GOAL_PLAN_REJECT_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, out)
}

func (h *Handler) goalFor(c *gin.Context) *goal.Service {
	if h.goal == nil {
		return nil
	}
	return h.goal.WithContext(c.Request.Context())
}
