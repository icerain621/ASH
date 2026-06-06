package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ash-repwiki/ash/internal/releases"
	"github.com/ash-repwiki/ash/internal/store"
)

type createReleaseRequest struct {
	Version        string `json:"version" binding:"required"`
	Title          string `json:"title,omitempty"`
	CanaryStrategy string `json:"canaryStrategy,omitempty"`
	SpaceID        string `json:"spaceId,omitempty"`
}

type ReleaseListResponse struct {
	Items []store.ReleaseRecord `json:"items"`
}

type ReleaseChecklistResponse struct {
	Items []store.ReleaseChecklistItem `json:"items"`
}

type patchReleaseChecklistRequest struct {
	Items []releases.ChecklistUpdate `json:"items"`
}

type createRollbackDrillRequest struct {
	Scenario     string   `json:"scenario" binding:"required"`
	Status       string   `json:"status,omitempty"`
	DurationMs   int64    `json:"durationMs,omitempty"`
	EvidenceRefs []string `json:"evidenceRefs,omitempty"`
	Notes        string   `json:"notes,omitempty"`
}

// CreateRelease godoc
// @Summary Create a release governance record
// @Tags releases
// @Accept json
// @Produce json
// @Param body body createReleaseRequest true "release"
// @Success 201 {object} store.ReleaseRecord
// @Failure 400 {object} APIErrorResponse
// @Failure 403 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/releases [post]
func (h *Handler) createRelease(c *gin.Context) {
	var req createReleaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", err.Error()))
		return
	}
	space := firstNonEmptyAPI(req.SpaceID, currentSpace(c))
	if !h.requireTargetSpace(c, space) {
		return
	}
	if !h.requirePermission(c, permReleaseWrite, space) {
		return
	}
	row, err := h.releasesFor(c).Create(releases.CreateRequest{
		SpaceID: space, Version: req.Version, Title: req.Title,
		CanaryStrategy: req.CanaryStrategy, CreatedBy: currentActor(c),
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("RELEASE_CREATE_FAILED", err.Error()))
		return
	}
	_ = h.dbFor(c).Create(auditRow(space, currentActor(c), "release.created", map[string]any{
		"releaseId": row.ID, "version": row.Version,
	}))
	c.JSON(http.StatusCreated, row)
}

// ListReleases godoc
// @Summary List release governance records
// @Tags releases
// @Produce json
// @Param limit query int false "max items" default(50)
// @Success 200 {object} ReleaseListResponse
// @Failure 403 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/releases [get]
func (h *Handler) listReleases(c *gin.Context) {
	space := currentSpace(c)
	if !h.requirePermission(c, permReleaseRead, space) {
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	rows, err := h.releasesFor(c).List(space, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("RELEASE_LIST_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, ReleaseListResponse{Items: rows})
}

// GetReleaseChecklist godoc
// @Summary Get release checklist
// @Tags releases
// @Produce json
// @Param releaseId path string true "release id"
// @Success 200 {object} ReleaseChecklistResponse
// @Failure 403 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/releases/{releaseId}/checklist [get]
func (h *Handler) getReleaseChecklist(c *gin.Context) {
	space := currentSpace(c)
	if !h.requirePermission(c, permReleaseRead, space) {
		return
	}
	rows, err := h.releasesFor(c).Checklist(space, c.Param("releaseId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("RELEASE_CHECKLIST_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, ReleaseChecklistResponse{Items: rows})
}

// PatchReleaseChecklist godoc
// @Summary Update release checklist items
// @Tags releases
// @Accept json
// @Produce json
// @Param releaseId path string true "release id"
// @Param body body patchReleaseChecklistRequest true "checklist patch"
// @Success 200 {object} ReleaseChecklistResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 403 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/releases/{releaseId}/checklist [patch]
func (h *Handler) patchReleaseChecklist(c *gin.Context) {
	space := currentSpace(c)
	if !h.requirePermission(c, permReleaseWrite, space) {
		return
	}
	var req patchReleaseChecklistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", err.Error()))
		return
	}
	rows, err := h.releasesFor(c).PatchChecklist(space, c.Param("releaseId"), currentActor(c), req.Items)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("RELEASE_CHECKLIST_UPDATE_FAILED", err.Error()))
		return
	}
	_ = h.dbFor(c).Create(auditRow(space, currentActor(c), "release.checklist_updated", map[string]any{
		"releaseId": c.Param("releaseId"), "count": len(req.Items),
	}))
	c.JSON(http.StatusOK, ReleaseChecklistResponse{Items: rows})
}

// EvaluateReleaseGate godoc
// @Summary Evaluate release gates
// @Tags releases
// @Produce json
// @Param releaseId path string true "release id"
// @Success 200 {object} releases.GateEvaluation
// @Failure 403 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/releases/{releaseId}/gate [post]
func (h *Handler) evaluateReleaseGate(c *gin.Context) {
	space := currentSpace(c)
	if !h.requirePermission(c, permReleaseWrite, space) {
		return
	}
	resp, err := h.releasesFor(c).EvaluateGate(space, c.Param("releaseId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("RELEASE_GATE_FAILED", err.Error()))
		return
	}
	_ = h.dbFor(c).Create(auditRow(space, currentActor(c), "release.gate_evaluated", map[string]any{
		"releaseId": resp.ReleaseID, "overall": resp.Overall, "results": len(resp.Results),
	}))
	c.JSON(http.StatusOK, resp)
}

// CreateRollbackDrill godoc
// @Summary Record a rollback drill
// @Tags releases
// @Accept json
// @Produce json
// @Param releaseId path string true "release id"
// @Param body body createRollbackDrillRequest true "rollback drill"
// @Success 201 {object} store.RollbackDrill
// @Failure 400 {object} APIErrorResponse
// @Failure 403 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/releases/{releaseId}/rollback-drills [post]
func (h *Handler) createRollbackDrill(c *gin.Context) {
	space := currentSpace(c)
	if !h.requirePermission(c, permReleaseWrite, space) {
		return
	}
	var req createRollbackDrillRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", err.Error()))
		return
	}
	row, err := h.releasesFor(c).CreateRollbackDrill(releases.RollbackDrillRequest{
		SpaceID: space, ReleaseID: c.Param("releaseId"), Scenario: req.Scenario,
		Status: req.Status, DurationMs: req.DurationMs, EvidenceRefs: req.EvidenceRefs,
		Notes: req.Notes, CreatedBy: currentActor(c),
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("ROLLBACK_DRILL_CREATE_FAILED", err.Error()))
		return
	}
	_ = h.dbFor(c).Create(auditRow(space, currentActor(c), "release.rollback_drill_recorded", map[string]any{
		"releaseId": row.ReleaseID, "drillId": row.ID, "status": row.Status,
	}))
	c.JSON(http.StatusCreated, row)
}
