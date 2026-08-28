package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/ash-repwiki/ash/internal/harness"
)

type harnessListResponse struct {
	Items []harness.ProfileView `json:"items"`
}

type harnessLoadActiveResponse struct {
	Profile harness.ProfileView `json:"profile"`
}

// ListHarnessProfiles godoc
// @Summary List harness profiles
// @Tags harness
// @Produce json
// @Param status query string false "draft|in_review|active|archived"
// @Param name query string false "profile name"
// @Success 200 {object} harnessListResponse
// @Router /api/v1/harness/profiles [get]
func (h *Handler) listHarnessProfiles(c *gin.Context) {
	items, err := h.harnessFor(c).List(currentSpace(c), c.Query("status"), c.Query("name"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("HARNESS_LIST_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, harnessListResponse{Items: items})
}

// CreateHarnessProfile godoc
// @Summary Create draft harness profile
// @Tags harness
// @Accept json
// @Produce json
// @Param body body harness.CreateRequest true "profile"
// @Success 201 {object} harness.ProfileView
// @Failure 400 {object} APIErrorResponse
// @Router /api/v1/harness/profiles [post]
func (h *Handler) createHarnessProfile(c *gin.Context) {
	var req harness.CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", err.Error()))
		return
	}
	req.SpaceID = firstNonEmptyAPI(strings.TrimSpace(req.SpaceID), currentSpace(c))
	if !h.requireTargetSpace(c, req.SpaceID) {
		return
	}
	if req.CreatedBy == "" {
		req.CreatedBy = currentActor(c)
	}
	view, err := h.harnessFor(c).Create(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("HARNESS_CREATE_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusCreated, view)
}

// GetHarnessProfile godoc
// @Summary Get harness profile by id
// @Tags harness
// @Produce json
// @Param profileId path string true "profile id"
// @Success 200 {object} harness.ProfileView
// @Failure 404 {object} APIErrorResponse
// @Router /api/v1/harness/profiles/{profileId} [get]
func (h *Handler) getHarnessProfile(c *gin.Context) {
	view, err := h.harnessFor(c).Get(c.Param("profileId"))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, errorBody("HARNESS_NOT_FOUND", "profile not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, errorBody("HARNESS_GET_FAILED", err.Error()))
		return
	}
	if !h.requireRequestSpace(c, view.SpaceID) {
		return
	}
	c.JSON(http.StatusOK, view)
}

// UpdateHarnessProfile godoc
// @Summary Update draft harness profile
// @Tags harness
// @Accept json
// @Produce json
// @Param profileId path string true "profile id"
// @Param body body harness.UpdateRequest true "spec"
// @Success 200 {object} harness.ProfileView
// @Failure 400 {object} APIErrorResponse
// @Router /api/v1/harness/profiles/{profileId} [put]
func (h *Handler) updateHarnessProfile(c *gin.Context) {
	view, err := h.harnessFor(c).Get(c.Param("profileId"))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, errorBody("HARNESS_NOT_FOUND", "profile not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, errorBody("HARNESS_GET_FAILED", err.Error()))
		return
	}
	if !h.requireRequestSpace(c, view.SpaceID) {
		return
	}
	var req harness.UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", err.Error()))
		return
	}
	out, err := h.harnessFor(c).Update(c.Param("profileId"), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("HARNESS_UPDATE_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, out)
}

// SubmitHarnessProfileReview godoc
// @Summary Submit harness profile for orchestration review
// @Tags harness
// @Produce json
// @Param profileId path string true "profile id"
// @Success 200 {object} harness.ProfileView
// @Router /api/v1/harness/profiles/{profileId}/submit-review [post]
func (h *Handler) submitHarnessProfileReview(c *gin.Context) {
	view, err := h.harnessFor(c).Get(c.Param("profileId"))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, errorBody("HARNESS_NOT_FOUND", "profile not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, errorBody("HARNESS_GET_FAILED", err.Error()))
		return
	}
	if !h.requireRequestSpace(c, view.SpaceID) {
		return
	}
	out, err := h.harnessFor(c).SubmitReview(c.Param("profileId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("HARNESS_SUBMIT_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, out)
}

// PromoteHarnessProfile godoc
// @Summary Promote harness profile to active
// @Tags harness
// @Produce json
// @Param profileId path string true "profile id"
// @Success 200 {object} harness.ProfileView
// @Router /api/v1/harness/profiles/{profileId}/promote [post]
func (h *Handler) promoteHarnessProfile(c *gin.Context) {
	view, err := h.harnessFor(c).Get(c.Param("profileId"))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, errorBody("HARNESS_NOT_FOUND", "profile not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, errorBody("HARNESS_GET_FAILED", err.Error()))
		return
	}
	if !h.requireRequestSpace(c, view.SpaceID) {
		return
	}
	out, err := h.harnessFor(c).Promote(c.Param("profileId"), currentActor(c))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("HARNESS_PROMOTE_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, out)
}

// RollbackHarnessProfile godoc
// @Summary Rollback active harness profile to previous archived version
// @Tags harness
// @Produce json
// @Param profileId path string true "active profile id"
// @Success 200 {object} harness.ProfileView
// @Router /api/v1/harness/profiles/{profileId}/rollback [post]
func (h *Handler) rollbackHarnessProfile(c *gin.Context) {
	view, err := h.harnessFor(c).Get(c.Param("profileId"))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, errorBody("HARNESS_NOT_FOUND", "profile not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, errorBody("HARNESS_GET_FAILED", err.Error()))
		return
	}
	if !h.requireRequestSpace(c, view.SpaceID) {
		return
	}
	out, err := h.harnessFor(c).Rollback(c.Param("profileId"), currentActor(c))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("HARNESS_ROLLBACK_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, out)
}

// LoadActiveHarnessProfile godoc
// @Summary Load active harness profile (platform default if missing)
// @Tags harness
// @Produce json
// @Param name query string false "profile name" default(default)
// @Success 200 {object} harnessLoadActiveResponse
// @Router /api/v1/harness/profiles/active [get]
func (h *Handler) loadActiveHarnessProfile(c *gin.Context) {
	name := c.DefaultQuery("name", "default")
	view, err := h.harnessFor(c).LoadActive(currentSpace(c), name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("HARNESS_LOAD_ACTIVE_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, harnessLoadActiveResponse{Profile: *view})
}

func (h *Handler) harnessFor(c *gin.Context) *harness.Service {
	return h.harness.WithContext(c.Request.Context())
}
