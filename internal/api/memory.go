package api

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/ash-repwiki/ash/internal/memory"
	"github.com/ash-repwiki/ash/internal/store"
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
	spaceID := currentSpace(c)
	if req.RunID != "" {
		if !h.requireRunAccess(c, req.RunID) {
			return
		}
		var err error
		spaceID, err = h.runSpaceID(c, req.RunID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errorBody("RUN_SCOPE_CHECK_FAILED", err.Error()))
			return
		}
	}
	if !h.requirePermission(c, permMemoryCreate, spaceID) {
		return
	}
	req.SpaceID = spaceID
	if req.ActorID == "" {
		req.ActorID = currentActor(c)
	}
	resp, err := h.memory.WithContext(c.Request.Context()).CreateCandidate(req)
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
	if !h.requirePermission(c, permMemoryRead) {
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	resp, err := h.memory.WithContext(c.Request.Context()).ListCandidatesForSpace(
		currentSpace(c),
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
	spaceID, err := h.memoryRecordSpace(c, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, errorBody("MEMORY_NOT_FOUND", memory.ErrNotFound.Error()))
			return
		}
		c.JSON(http.StatusInternalServerError, errorBody("MEMORY_SCOPE_CHECK_FAILED", err.Error()))
		return
	}
	if !h.requireRequestSpace(c, spaceID) {
		return
	}
	if !h.requirePermission(c, permMemoryReview, spaceID) {
		return
	}
	if req.ReviewerID == "" {
		req.ReviewerID = currentActor(c)
	}
	if req.ActorID == "" {
		req.ActorID = currentActor(c)
	}
	resp, err := h.memory.WithContext(c.Request.Context()).Review(id, req)
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
	if !h.requirePermission(c, permMemoryQuery) {
		return
	}
	resp, err := h.memory.WithContext(c.Request.Context()).QueryForSpace(currentSpace(c), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("MEMORY_QUERY_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, resp)
}

// RunMemoryMigration godoc
// @Summary Apply pending memory schema migrations
// @Tags memory
// @Accept json
// @Produce json
// @Param body body memory.RunMigrationRequest true "migration"
// @Success 200 {object} memory.RunMigrationResponse
// @Failure 400 {object} APIErrorResponse
// @Router /api/v1/memory/migrate [post]
func (h *Handler) runMemoryMigration(c *gin.Context) {
	var req memory.RunMigrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", err.Error()))
		return
	}
	if req.RunID != "" {
		if !h.requireRunAccess(c, req.RunID) {
			return
		}
	}
	if !h.requirePermission(c, permMemoryReview) {
		return
	}
	resp, err := h.memory.WithContext(c.Request.Context()).RunMigrations(req)
	if err != nil {
		if errors.Is(err, memory.ErrRunNotFound) {
			c.JSON(http.StatusNotFound, errorBody("RUN_NOT_FOUND", err.Error()))
			return
		}
		c.JSON(http.StatusBadRequest, errorBody("MEMORY_MIGRATION_FAILED", err.Error()))
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
	if !h.requireRunAccess(c, req.RunID) {
		return
	}
	spaceID, err := h.runSpaceID(c, req.RunID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("RUN_SCOPE_CHECK_FAILED", err.Error()))
		return
	}
	if !h.requirePermission(c, permMemoryUse, spaceID) {
		return
	}
	if ok, err := h.memoryRecordsInSpace(c, req.RecordIDs, spaceID); err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("MEMORY_SCOPE_CHECK_FAILED", err.Error()))
		return
	} else if !ok {
		c.JSON(http.StatusNotFound, errorBody("MEMORY_NOT_FOUND", "memory records not found in run space"))
		return
	}
	if req.ActorID == "" {
		req.ActorID = currentActor(c)
	}
	resp, err := h.memory.WithContext(c.Request.Context()).HitUsed(req)
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
	id := c.Param("recordId")
	spaceID, err := h.memoryRecordSpace(c, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, errorBody("MEMORY_NOT_FOUND", memory.ErrNotFound.Error()))
			return
		}
		c.JSON(http.StatusInternalServerError, errorBody("MEMORY_SCOPE_CHECK_FAILED", err.Error()))
		return
	}
	if err := store.EnforceSpaceAccess(spaceID, currentSpace(c)); err != nil {
		c.JSON(http.StatusForbidden, errorBody("SPACE_ACCESS_DENIED", err.Error()))
		return
	}
	if !h.requirePermission(c, permMemoryRead, spaceID) {
		return
	}
	rec, err := h.memory.WithContext(c.Request.Context()).GetForSpace(spaceID, id)
	if err != nil {
		c.JSON(http.StatusNotFound, errorBody("MEMORY_NOT_FOUND", err.Error()))
		return
	}
	c.JSON(http.StatusOK, rec)
}

func (h *Handler) memoryRecordSpace(c *gin.Context, id string) (string, error) {
	var row store.MemoryRecord
	if err := h.dbFor(c).Select("space_id").First(&row, "id = ?", id).Error; err != nil {
		return "", err
	}
	return firstNonEmptyAPI(row.SpaceID, "local"), nil
}

func (h *Handler) memoryRecordsInSpace(c *gin.Context, ids []string, spaceID string) (bool, error) {
	if len(ids) == 0 {
		return false, nil
	}
	var count int64
	if err := h.dbFor(c).Model(&store.MemoryRecord{}).
		Where("id IN ? AND space_id = ?", ids, firstNonEmptyAPI(spaceID, "local")).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count == int64(len(ids)), nil
}
