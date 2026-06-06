package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/ash-repwiki/ash/internal/doctor"
	"github.com/ash-repwiki/ash/internal/runs"
)

type doctorRunRequest struct {
	Suite  string `json:"suite" binding:"required"`
	Format string `json:"format"`
}

type doctorRunResponse struct {
	ReportID string `json:"reportId"`
}

type reportStore struct {
	mu   sync.RWMutex
	byID map[string]*doctor.Report
}

func newReportStore() *reportStore {
	return &reportStore{byID: make(map[string]*doctor.Report)}
}

func (s *reportStore) put(rep *doctor.Report) string {
	id := "drpt_" + uuid.NewString()
	s.mu.Lock()
	s.byID[id] = rep
	s.mu.Unlock()
	return id
}

func (s *reportStore) get(id string) (*doctor.Report, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rep, ok := s.byID[id]
	return rep, ok
}

func (h *Handler) ensureDoctor() {
	if h.doctorReports == nil {
		h.doctorReports = newReportStore()
	}
}

// RunDoctor godoc
// @Summary Run validation suite
// @Tags doctor
// @Accept json
// @Produce json
// @Param body body doctorRunRequest true "suite request"
// @Success 200 {object} doctorRunResponse
// @Failure 400 {object} APIErrorResponse
// @Router /api/v1/doctor/run [post]
func (h *Handler) runDoctor(c *gin.Context) {
	h.ensureDoctor()
	var req doctorRunRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", err.Error()))
		return
	}
	suite := req.Suite
	rep, err := h.doctor.RunSuite(suite)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("DOCTOR_FAILED", err.Error()))
		return
	}
	id := h.doctorReports.put(rep)
	if req.Format == "md" {
		_ = writeReportMD(h.db.DataDir(), id, rep)
	}
	_ = h.db.Create(auditRow(currentSpace(c), currentActor(c), "doctor.suite_completed", map[string]any{
		"reportId": id, "suite": rep.Suite, "pass": rep.Summary.Pass, "fail": rep.Summary.Fail,
	}))
	c.JSON(http.StatusOK, doctorRunResponse{ReportID: id})
}

// GetDoctorReport godoc
// @Summary Get doctor report
// @Tags doctor
// @Produce json
// @Param reportId path string true "report id"
// @Success 200 {object} doctor.Report
// @Failure 404 {object} APIErrorResponse
// @Router /api/v1/doctor/reports/{reportId} [get]
func (h *Handler) getDoctorReport(c *gin.Context) {
	h.ensureDoctor()
	id := c.Param("reportId")
	rep, ok := h.doctorReports.get(id)
	if !ok {
		c.JSON(http.StatusNotFound, errorBody("REPORT_NOT_FOUND", "report not found"))
		return
	}
	c.JSON(http.StatusOK, rep)
}

// ListRunArtifacts godoc
// @Summary List run artifacts manifest
// @Tags runs
// @Produce json
// @Param runId path string true "run id"
// @Success 200 {object} artifactsManifestResponse
// @Failure 404 {object} APIErrorResponse
// @Router /api/v1/runs/{runId}/artifacts [get]
func (h *Handler) listRunArtifacts(c *gin.Context) {
	runID := c.Param("runId")
	if !h.requireRunPermission(c, runID, permArtifactRead) {
		return
	}
	manifest, err := h.runs.Artifacts(runID)
	if err != nil {
		c.JSON(http.StatusNotFound, errorBody("ARTIFACTS_NOT_FOUND", err.Error()))
		return
	}
	c.JSON(http.StatusOK, artifactsManifestResponse{Manifest: manifest, Artifacts: manifest.Artifacts})
}

// ListRunCheckpoints godoc
// @Summary List run checkpoints
// @Tags runs
// @Produce json
// @Param runId path string true "run id"
// @Success 200 {object} CheckpointListResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/runs/{runId}/checkpoints [get]
func (h *Handler) listRunCheckpoints(c *gin.Context) {
	runID := c.Param("runId")
	if !h.requireRunPermission(c, runID, permArtifactRead) {
		return
	}
	items, err := h.runs.Checkpoints(runID)
	if err != nil {
		c.JSON(http.StatusNotFound, errorBody("CHECKPOINT_LIST_NOT_FOUND", err.Error()))
		return
	}
	c.JSON(http.StatusOK, CheckpointListResponse{Items: items})
}

// GetRunArtifactAccess godoc
// @Summary Get a signed artifact access URL
// @Tags runs
// @Produce json
// @Param runId path string true "run id"
// @Param artifactName path string true "artifact name"
// @Param ttlSeconds query int false "signed URL ttl seconds" default(900)
// @Success 200 {object} ArtifactAccessResponse
// @Failure 404 {object} APIErrorResponse
// @Router /api/v1/runs/{runId}/artifacts/{artifactName}/access [get]
func (h *Handler) getRunArtifactAccess(c *gin.Context) {
	runID := c.Param("runId")
	if !h.requireRunPermission(c, runID, permArtifactRead) {
		return
	}
	ttl := 15 * time.Minute
	if raw := c.Query("ttlSeconds"); raw != "" {
		if seconds, err := strconv.Atoi(raw); err == nil && seconds > 0 {
			ttl = time.Duration(seconds) * time.Second
		}
	}
	resp, err := h.runs.ArtifactAccess(runID, c.Param("artifactName"), ttl)
	if err != nil {
		c.JSON(http.StatusNotFound, errorBody("ARTIFACT_ACCESS_NOT_FOUND", err.Error()))
		return
	}
	c.JSON(http.StatusOK, resp)
}

// GetRunCheckpointAccess godoc
// @Summary Get a signed checkpoint access URL
// @Tags runs
// @Produce json
// @Param runId path string true "run id"
// @Param checkpointId path string true "checkpoint id"
// @Param ttlSeconds query int false "signed URL ttl seconds" default(900)
// @Success 200 {object} CheckpointAccessResponse
// @Failure 404 {object} APIErrorResponse
// @Router /api/v1/runs/{runId}/checkpoints/{checkpointId}/access [get]
func (h *Handler) getRunCheckpointAccess(c *gin.Context) {
	runID := c.Param("runId")
	if !h.requireRunPermission(c, runID, permArtifactRead) {
		return
	}
	ttl := 15 * time.Minute
	if raw := c.Query("ttlSeconds"); raw != "" {
		if seconds, err := strconv.Atoi(raw); err == nil && seconds > 0 {
			ttl = time.Duration(seconds) * time.Second
		}
	}
	resp, err := h.runs.CheckpointAccess(runID, c.Param("checkpointId"), ttl)
	if err != nil {
		c.JSON(http.StatusNotFound, errorBody("CHECKPOINT_ACCESS_NOT_FOUND", err.Error()))
		return
	}
	c.JSON(http.StatusOK, resp)
}

type artifactsManifestResponse struct {
	Manifest  any `json:"manifest"`
	Artifacts any `json:"artifacts"`
}

type ArtifactAccessResponse = runs.ArtifactAccessResponse
type CheckpointAccessResponse = runs.CheckpointAccessResponse

func writeReportMD(dataDir, id string, rep *doctor.Report) error {
	dir := filepath.Join(dataDir, "doctor")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	b, _ := json.MarshalIndent(rep, "", "  ")
	path := filepath.Join(dir, id+".json")
	return os.WriteFile(path, b, 0o644)
}
