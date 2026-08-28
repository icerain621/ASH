package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/ash-repwiki/ash/internal/scenariopatch"
)

type scenarioPatchListResponse struct {
	Items []scenariopatch.View `json:"items"`
}

// ListScenarioPatches godoc
// @Summary List scenario patch drafts
// @Tags reviews
// @Produce json
// @Param status query string false "draft|in_review|approved|rejected|archived"
// @Success 200 {object} scenarioPatchListResponse
// @Router /api/v1/scenario-patches [get]
func (h *Handler) listScenarioPatches(c *gin.Context) {
	space := currentSpace(c)
	if !h.requirePermission(c, permFeedbackRead, space) {
		return
	}
	items, err := h.patches.WithContext(c.Request.Context()).List(space, c.Query("status"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("SCENARIO_PATCH_LIST_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, scenarioPatchListResponse{Items: items})
}

// CreateScenarioPatch godoc
// @Summary Create scenario patch draft
// @Tags reviews
// @Accept json
// @Produce json
// @Param body body scenariopatch.CreateRequest true "patch"
// @Success 201 {object} scenariopatch.View
// @Router /api/v1/scenario-patches [post]
func (h *Handler) createScenarioPatch(c *gin.Context) {
	var req scenariopatch.CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", err.Error()))
		return
	}
	req.SpaceID = firstNonEmptyAPI(strings.TrimSpace(req.SpaceID), currentSpace(c))
	if !h.requireTargetSpace(c, req.SpaceID) {
		return
	}
	if !h.requirePermission(c, permFeedbackWrite, req.SpaceID) {
		return
	}
	if req.CreatedBy == "" {
		req.CreatedBy = currentActor(c)
	}
	view, err := h.patches.WithContext(c.Request.Context()).Create(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("SCENARIO_PATCH_CREATE_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusCreated, view)
}

// SubmitScenarioPatchReview godoc
// @Summary Submit scenario patch for orchestration review
// @Tags reviews
// @Produce json
// @Param patchId path string true "patch id"
// @Success 200 {object} scenariopatch.View
// @Router /api/v1/scenario-patches/{patchId}/submit-review [post]
func (h *Handler) submitScenarioPatchReview(c *gin.Context) {
	view, err := h.patches.WithContext(c.Request.Context()).Get(c.Param("patchId"))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, errorBody("SCENARIO_PATCH_NOT_FOUND", "patch not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, errorBody("SCENARIO_PATCH_SUBMIT_FAILED", err.Error()))
		return
	}
	if !h.requireRequestSpace(c, view.SpaceID) {
		return
	}
	if !h.requirePermission(c, permFeedbackWrite, view.SpaceID) {
		return
	}
	out, err := h.patches.WithContext(c.Request.Context()).SubmitReview(c.Param("patchId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("SCENARIO_PATCH_SUBMIT_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, out)
}
