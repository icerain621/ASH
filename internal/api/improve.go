package api

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ash-repwiki/ash/internal/improve"
	"github.com/ash-repwiki/ash/internal/runs"
)

// CreateImproveProposal godoc
// @Summary Create self-improvement proposal
// @Tags improve
// @Accept json
// @Produce json
// @Param body body improve.CreateProposalRequest true "proposal"
// @Success 201 {object} improve.ProposalView
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Router /api/v1/improve/proposals [post]
func (h *Handler) createImproveProposal(c *gin.Context) {
	var req improve.CreateProposalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", err.Error()))
		return
	}
	spaceID := currentSpace(c)
	if req.BaselineRunID != "" && !h.requireRunAccess(c, req.BaselineRunID) {
		return
	}
	req.SpaceID = spaceID
	if req.ActorID == "" {
		req.ActorID = currentActor(c)
	}
	resp, err := h.improve.Create(req)
	if err != nil {
		if errors.Is(err, runs.ErrRunNotFound) {
			c.JSON(http.StatusNotFound, errorBody("RUN_NOT_FOUND", err.Error()))
			return
		}
		if errors.Is(err, improve.ErrBaselineNotReady) {
			c.JSON(http.StatusBadRequest, errorBody("BASELINE_NOT_READY", err.Error()))
			return
		}
		c.JSON(http.StatusBadRequest, errorBody("IMPROVE_CREATE_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusCreated, resp)
}

// ListImproveProposals godoc
// @Summary List improvement proposals
// @Tags improve
// @Produce json
// @Param limit query int false "max items" default(50)
// @Success 200 {object} improve.ListProposalsResponse
// @Router /api/v1/improve/proposals [get]
func (h *Handler) listImproveProposals(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	resp, err := h.improve.List(currentSpace(c), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("IMPROVE_LIST_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, resp)
}

// GetImproveProposal godoc
// @Summary Get improvement proposal
// @Tags improve
// @Produce json
// @Param proposalId path string true "proposal id"
// @Success 200 {object} improve.ProposalView
// @Failure 404 {object} APIErrorResponse
// @Router /api/v1/improve/proposals/{proposalId} [get]
func (h *Handler) getImproveProposal(c *gin.Context) {
	resp, err := h.improve.Get(currentSpace(c), c.Param("proposalId"))
	if err != nil {
		if errors.Is(err, improve.ErrNotFound) {
			c.JSON(http.StatusNotFound, errorBody("PROPOSAL_NOT_FOUND", err.Error()))
			return
		}
		c.JSON(http.StatusInternalServerError, errorBody("IMPROVE_GET_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, resp)
}

// StartImproveExperiment godoc
// @Summary Run experiment replay for proposal
// @Tags improve
// @Produce json
// @Param proposalId path string true "proposal id"
// @Success 200 {object} improve.StartExperimentResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Router /api/v1/improve/proposals/{proposalId}/experiment [post]
func (h *Handler) startImproveExperiment(c *gin.Context) {
	resp, err := h.improve.StartExperiment(currentSpace(c), c.Param("proposalId"))
	if err != nil {
		if errors.Is(err, improve.ErrNotFound) {
			c.JSON(http.StatusNotFound, errorBody("PROPOSAL_NOT_FOUND", err.Error()))
			return
		}
		if errors.Is(err, improve.ErrInvalidState) {
			c.JSON(http.StatusBadRequest, errorBody("INVALID_STATE", err.Error()))
			return
		}
		c.JSON(http.StatusBadRequest, errorBody("IMPROVE_EXPERIMENT_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, resp)
}

// StartImproveCanary godoc
// @Summary Start canary rollout for proposal
// @Tags improve
// @Accept json
// @Produce json
// @Param proposalId path string true "proposal id"
// @Param body body improve.CanaryRequest true "canary"
// @Success 200 {object} improve.ProposalView
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Router /api/v1/improve/proposals/{proposalId}/canary [post]
func (h *Handler) startImproveCanary(c *gin.Context) {
	var req improve.CanaryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", err.Error()))
		return
	}
	resp, err := h.improve.StartCanary(currentSpace(c), c.Param("proposalId"), req)
	if err != nil {
		if errors.Is(err, improve.ErrNotFound) {
			c.JSON(http.StatusNotFound, errorBody("PROPOSAL_NOT_FOUND", err.Error()))
			return
		}
		c.JSON(http.StatusBadRequest, errorBody("IMPROVE_CANARY_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, resp)
}

// PromoteImproveProposal godoc
// @Summary Promote proposal after canary
// @Tags improve
// @Produce json
// @Param proposalId path string true "proposal id"
// @Success 200 {object} improve.StatusResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Router /api/v1/improve/proposals/{proposalId}/promote [post]
func (h *Handler) promoteImproveProposal(c *gin.Context) {
	resp, err := h.improve.Promote(currentSpace(c), c.Param("proposalId"))
	if err != nil {
		if errors.Is(err, improve.ErrNotFound) {
			c.JSON(http.StatusNotFound, errorBody("PROPOSAL_NOT_FOUND", err.Error()))
			return
		}
		c.JSON(http.StatusBadRequest, errorBody("IMPROVE_PROMOTE_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, resp)
}

// RollbackImproveProposal godoc
// @Summary Rollback promoted proposal
// @Tags improve
// @Produce json
// @Param proposalId path string true "proposal id"
// @Success 200 {object} improve.StatusResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Router /api/v1/improve/proposals/{proposalId}/rollback [post]
func (h *Handler) rollbackImproveProposal(c *gin.Context) {
	resp, err := h.improve.Rollback(currentSpace(c), c.Param("proposalId"))
	if err != nil {
		if errors.Is(err, improve.ErrNotFound) {
			c.JSON(http.StatusNotFound, errorBody("PROPOSAL_NOT_FOUND", err.Error()))
			return
		}
		c.JSON(http.StatusBadRequest, errorBody("IMPROVE_ROLLBACK_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, resp)
}
