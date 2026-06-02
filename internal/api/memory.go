package api

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ash-repwiki/ash/internal/memory"
)

// CreateMemoryCandidate godoc
// @Summary Create memory candidate
// @Tags memory
// @Accept json
// @Produce json
// @Param body body memory.CreateCandidateRequest true "candidate"
// @Success 201 {object} memory.CreateCandidateResponse
// @Failure 400 {object} APIErrorResponse
// @Router /api/v1/memory/candidates [post]
func (h *Handler) createMemoryCandidate(c *gin.Context) {
	var req memory.CreateCandidateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", err.Error()))
		return
	}
	resp, err := h.memory.CreateCandidate(req)
	if err != nil {
		if errors.Is(err, memory.ErrRunNotFound) {
			c.JSON(http.StatusNotFound, errorBody("RUN_NOT_FOUND", err.Error()))
			return
		}
		c.JSON(http.StatusBadRequest, errorBody("MEMORY_CREATE_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusCreated, resp)
}

// ListMemoryCandidates godoc
// @Summary List memory candidates
// @Tags memory
// @Produce json
// @Param layer query string false "layer filter L0-L3"
// @Param status query string false "status filter" default(candidate)
// @Param repo query string false "scope repo"
// @Param limit query int false "page size" default(50)
// @Param offset query int false "offset" default(0)
// @Success 200 {object} memory.ListCandidatesResponse
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/memory/candidates [get]
func (h *Handler) listMemoryCandidates(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	resp, err := h.memory.ListCandidates(
		c.Query("layer"),
		c.Query("status"),
		c.Query("repo"),
		limit,
		offset,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("MEMORY_LIST_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, resp)
}

// ReviewMemoryCandidate godoc
// @Summary Review a memory candidate
// @Tags memory
// @Accept json
// @Produce json
// @Param candidateId path string true "candidate id"
// @Param body body memory.ReviewRequest true "review"
// @Success 200 {object} memory.ReviewResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Router /api/v1/memory/candidates/{candidateId}/review [post]
func (h *Handler) reviewMemoryCandidate(c *gin.Context) {
	id := c.Param("candidateId")
	var req memory.ReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", err.Error()))
		return
	}
	resp, err := h.memory.Review(id, req)
	if err != nil {
		if errors.Is(err, memory.ErrNotFound) {
			c.JSON(http.StatusNotFound, errorBody("MEMORY_NOT_FOUND", err.Error()))
			return
		}
		if errors.Is(err, memory.ErrRunNotFound) {
			c.JSON(http.StatusNotFound, errorBody("RUN_NOT_FOUND", err.Error()))
			return
		}
		c.JSON(http.StatusBadRequest, errorBody("MEMORY_REVIEW_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, resp)
}

// QueryMemory godoc
// @Summary Query approved memory records
// @Tags memory
// @Accept json
// @Produce json
// @Param body body memory.QueryRequest true "query"
// @Success 200 {object} memory.QueryResponse
// @Failure 400 {object} APIErrorResponse
// @Router /api/v1/memory/query [post]
func (h *Handler) queryMemory(c *gin.Context) {
	var req memory.QueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", err.Error()))
		return
	}
	resp, err := h.memory.Query(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("MEMORY_QUERY_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, resp)
}

// MemoryHitUsed godoc
// @Summary Record memory usage in a run
// @Tags memory
// @Accept json
// @Produce json
// @Param body body memory.HitUsedRequest true "hit used"
// @Success 200 {object} memory.HitUsedResponse
// @Failure 400 {object} APIErrorResponse
// @Router /api/v1/memory/hit-used [post]
func (h *Handler) memoryHitUsed(c *gin.Context) {
	var req memory.HitUsedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", err.Error()))
		return
	}
	resp, err := h.memory.HitUsed(req)
	if err != nil {
		if errors.Is(err, memory.ErrRunNotFound) {
			c.JSON(http.StatusNotFound, errorBody("RUN_NOT_FOUND", err.Error()))
			return
		}
		c.JSON(http.StatusBadRequest, errorBody("MEMORY_HIT_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, resp)
}

// GetMemoryRecord godoc
// @Summary Get memory record by id
// @Tags memory
// @Produce json
// @Param recordId path string true "memory id"
// @Success 200 {object} memory.RecordView
// @Failure 404 {object} APIErrorResponse
// @Router /api/v1/memory/records/{recordId} [get]
func (h *Handler) getMemoryRecord(c *gin.Context) {
	rec, err := h.memory.Get(c.Param("recordId"))
	if err != nil {
		c.JSON(http.StatusNotFound, errorBody("MEMORY_NOT_FOUND", err.Error()))
		return
	}
	c.JSON(http.StatusOK, rec)
}
