package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/ash-repwiki/ash/internal/artifactstore"
	"github.com/ash-repwiki/ash/internal/config"
	"github.com/ash-repwiki/ash/internal/doctor"
	"github.com/ash-repwiki/ash/internal/security"
	"github.com/ash-repwiki/ash/internal/store"
)

type SecretScanResponse struct {
	SpaceID       string                `json:"spaceId"`
	Scanned       int                   `json:"scanned"`
	LeakCount     int                   `json:"leakCount"`
	RedactEnabled bool                  `json:"redactEnabled"`
	Findings      []security.LeakFinding `json:"findings"`
}

type ComplianceExportRequest struct {
	Suite    string `json:"suite"`
	ReportID string `json:"reportId,omitempty"`
}

type ComplianceExportResponse struct {
	ExportID string         `json:"exportId"`
	ReportID string         `json:"reportId,omitempty"`
	Suite    string         `json:"suite"`
	Report   *doctor.Report `json:"report,omitempty"`
}

// ComplianceSecretScan godoc
// @Summary Scan audit logs and run events for secret patterns
// @Tags compliance
// @Produce json
// @Param limit query int false "max items per source" default(200)
// @Success 200 {object} SecretScanResponse
// @Failure 403 {object} APIErrorResponse
// @Router /api/v1/compliance/secret-scan [get]
func (h *Handler) complianceSecretScan(c *gin.Context) {
	space := currentSpace(c)
	if !h.requirePermission(c, permAuditExport, space) {
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "200"))
	c.JSON(http.StatusOK, h.collectSecretScan(c, space, limit))
}

func buildSecretScan(space string, redactEnabled bool, items []struct{ Source, Ref, Text string }) SecretScanResponse {
	findings := security.ScanTexts(items)
	return SecretScanResponse{
		SpaceID: space, Scanned: len(items), LeakCount: len(findings),
		RedactEnabled: redactEnabled, Findings: findings,
	}
}

func (h *Handler) collectSecretScan(c *gin.Context, space string, limit int) SecretScanResponse {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	db := h.dbFor(c)
	var policy store.AuditPolicy
	redactEnabled := false
	if err := db.First(&policy, "space_id = ?", space).Error; err == nil {
		redactEnabled = policy.RedactPayload
	}
	var auditLogs []store.AuditLog
	_ = db.Where("space_id = ?", space).Order("created_at desc").Limit(limit).Find(&auditLogs).Error
	var runIDs []string
	_ = db.Model(&store.RunRecord{}).Where("space_id = ?", space).Pluck("id", &runIDs).Error
	var events []store.RunEvent
	if len(runIDs) > 0 {
		_ = db.Where("run_id IN ?", runIDs).Order("seq desc").Limit(limit).Find(&events).Error
	}
	items := make([]struct{ Source, Ref, Text string }, 0, len(auditLogs)+len(events))
	for _, row := range auditLogs {
		items = append(items, struct{ Source, Ref, Text string }{
			Source: "audit_log", Ref: row.ID, Text: row.PayloadJSON,
		})
	}
	for _, row := range events {
		items = append(items, struct{ Source, Ref, Text string }{
			Source: "run_event", Ref: row.ID, Text: row.PayloadJSON,
		})
	}
	return buildSecretScan(space, redactEnabled, items)
}

// ComplianceExportBundle godoc
// @Summary Export audit bundle with doctor report and secret scan summary
// @Tags compliance
// @Accept json
// @Produce json
// @Param body body ComplianceExportRequest true "export request"
// @Success 202 {object} ComplianceExportResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 403 {object} APIErrorResponse
// @Router /api/v1/compliance/export [post]
func (h *Handler) complianceExportBundle(c *gin.Context) {
	space := currentSpace(c)
	if !h.requirePermission(c, permAuditExport, space) {
		return
	}
	var req ComplianceExportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", err.Error()))
		return
	}
	suite := strings.TrimSpace(req.Suite)
	if suite == "" {
		suite = "TR2"
	}
	h.ensureDoctor()
	var rep *doctor.Report
	var reportID string
	if req.ReportID != "" {
		var ok bool
		rep, ok = h.doctorReports.get(req.ReportID)
		if !ok {
			c.JSON(http.StatusNotFound, errorBody("REPORT_NOT_FOUND", "doctor report not found"))
			return
		}
		reportID = req.ReportID
	} else {
		var err error
		rep, err = h.doctor.RunSuite(suite)
		if err != nil {
			c.JSON(http.StatusBadRequest, errorBody("DOCTOR_FAILED", err.Error()))
			return
		}
		reportID = h.doctorReports.put(rep)
	}

	secretScan := h.collectSecretScan(c, space, 200)

	db := h.dbFor(c)
	var logs []store.AuditLog
	if err := db.Where("space_id = ?", space).Order("created_at asc").Find(&logs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("AUDIT_EXPORT_FAILED", err.Error()))
		return
	}
	var policy store.AuditPolicy
	if err := db.First(&policy, "space_id = ?", space).Error; err == nil && policy.RedactPayload {
		for i := range logs {
			logs[i].PayloadJSON = security.RedactJSON(logs[i].PayloadJSON)
		}
	}
	now := time.Now().UTC()
	row := store.AuditExport{
		ID: "audexp_" + uuid.NewString(), SpaceID: space,
		Status: "running", RequestedBy: currentActor(c), CreatedAt: now,
	}
	if err := db.Create(&row).Error; err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("AUDIT_EXPORT_CREATE_FAILED", err.Error()))
		return
	}
	payload := map[string]any{
		"exportId": row.ID, "spaceId": space, "createdAt": now.UnixMilli(),
		"doctorSuite": suite, "doctorReportId": reportID, "doctorReport": rep,
		"secretScan": secretScan, "logs": logs,
	}
	b, _ := json.MarshalIndent(payload, "", "  ")
	b = append(b, '\n')
	digest := sha256Digest(b)
	cfg := config.Load()
	artifactStore := artifactstore.New(cfg.ArtifactStore, h.db.DataDir())
	storeKey := "audit_exports/" + space + "/" + row.ID + ".json"
	ref, err := artifactStore.Put(context.Background(), storeKey, bytes.NewReader(b), "application/json")
	if err != nil {
		row.Status = "failed"
		_ = db.Save(&row).Error
		c.JSON(http.StatusInternalServerError, errorBody("AUDIT_EXPORT_FAILED", err.Error()))
		return
	}
	done := time.Now().UTC()
	row.Status = "completed"
	row.URI = ref.URI
	row.StoreKey = ref.Key
	row.Digest = digest
	row.ContentType = ref.ContentType
	row.SizeBytes = ref.SizeBytes
	row.CompletedAt = &done
	_ = db.Save(&row).Error
	_ = db.Create(auditRow(space, currentActor(c), "compliance.export_completed", map[string]any{
		"exportId": row.ID, "doctorReportId": reportID, "suite": suite, "digest": digest,
	})).Error
	c.JSON(http.StatusAccepted, ComplianceExportResponse{
		ExportID: row.ID, ReportID: reportID, Suite: suite, Report: rep,
	})
}
