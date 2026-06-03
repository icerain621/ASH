package api

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ash-repwiki/ash/internal/runs"
	"github.com/ash-repwiki/ash/internal/store"
)

type rejectApprovalRequest struct {
	ActorID string `json:"actorId,omitempty"`
	Reason  string `json:"reason,omitempty"`
}

// ListApprovals godoc
// @Summary List approval requests
// @Tags approvals
// @Produce json
// @Param status query string false "approval status" default(pending)
// @Param limit query int false "max items" default(50)
// @Success 200 {object} ApprovalRequestListResponse
// @Failure 403 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/approvals [get]
func (h *Handler) listApprovals(c *gin.Context) {
	space := currentSpace(c)
	if !h.requirePermission(c, permRunApprove, space) {
		return
	}
	status := strings.TrimSpace(c.DefaultQuery("status", "pending"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	q := h.db.Where("space_id = ?", space)
	if status != "" && status != "all" {
		q = q.Where("status = ?", status)
	}
	var rows []store.ApprovalRequest
	if err := q.Order("created_at desc").Limit(limit).Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("APPROVAL_LIST_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, ApprovalRequestListResponse{Items: rows})
}

// ApproveApproval godoc
// @Summary Approve an approval request
// @Tags approvals
// @Accept json
// @Produce json
// @Param approvalId path string true "approval id"
// @Param body body runs.ApproveRequest true "approval request"
// @Success 200 {object} runs.ApproveResponse
// @Failure 403 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/approvals/{approvalId}/approve [post]
func (h *Handler) approveApproval(c *gin.Context) {
	row, ok := h.lookupApproval(c)
	if !ok {
		return
	}
	if !h.requirePermission(c, permRunApprove, row.SpaceID) {
		return
	}
	var req runs.ApproveRequest
	_ = c.ShouldBindJSON(&req)
	if req.ActorID == "" {
		req.ActorID = currentActor(c)
	}
	resp, err := h.runs.Approve(row.RunID, req)
	if err != nil {
		writeRunControlError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// RejectApproval godoc
// @Summary Reject an approval request and cancel the run
// @Tags approvals
// @Accept json
// @Produce json
// @Param approvalId path string true "approval id"
// @Param body body rejectApprovalRequest true "rejection request"
// @Success 200 {object} runs.CancelResponse
// @Failure 403 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/approvals/{approvalId}/reject [post]
func (h *Handler) rejectApproval(c *gin.Context) {
	row, ok := h.lookupApproval(c)
	if !ok {
		return
	}
	if !h.requirePermission(c, permRunApprove, row.SpaceID) {
		return
	}
	var req rejectApprovalRequest
	_ = c.ShouldBindJSON(&req)
	actor := firstNonEmptyAPI(req.ActorID, currentActor(c))
	reason := firstNonEmptyAPI(req.Reason, "approval rejected")
	now := time.Now().UTC()
	if err := h.db.Model(&store.ApprovalRequest{}).
		Where("id = ? AND status = ?", row.ID, "pending").
		Updates(map[string]any{
			"status":          "rejected",
			"decided_by":      actor,
			"decision_reason": reason,
			"decided_at":      &now,
			"updated_at":      now,
		}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("APPROVAL_REJECT_FAILED", err.Error()))
		return
	}
	_, _ = h.events.Append(row.RunID, row.TraceID, "approval.rejected", "warn", map[string]any{
		"approvalId": row.ID, "actorId": actor, "reason": reason, "stepId": row.StepID,
	})
	_ = h.db.Create(auditRow(row.SpaceID, actor, "approval.rejected", map[string]any{
		"approvalId": row.ID, "runId": row.RunID, "stepId": row.StepID, "reason": reason,
	})).Error
	resp, err := h.runs.Cancel(row.RunID)
	if err != nil {
		writeRunControlError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) lookupApproval(c *gin.Context) (*store.ApprovalRequest, bool) {
	var row store.ApprovalRequest
	if err := h.db.First(&row, "id = ? AND space_id = ?", c.Param("approvalId"), currentSpace(c)).Error; err != nil {
		c.JSON(http.StatusNotFound, errorBody("APPROVAL_NOT_FOUND", "approval not found"))
		return nil, false
	}
	return &row, true
}
