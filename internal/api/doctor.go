package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/ash-repwiki/ash/internal/doctor"
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
	if suite == "ALL" {
		suite = "TR0"
	}
	rep, err := h.doctor.RunSuite(suite)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("DOCTOR_FAILED", err.Error()))
		return
	}
	id := h.doctorReports.put(rep)
	if req.Format == "md" {
		_ = writeReportMD(h.db.DataDir(), id, rep)
	}
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
	manifest, err := h.runs.Artifacts(runID)
	if err != nil {
		c.JSON(http.StatusNotFound, errorBody("ARTIFACTS_NOT_FOUND", err.Error()))
		return
	}
	c.JSON(http.StatusOK, artifactsManifestResponse{Manifest: manifest, Artifacts: manifest.Artifacts})
}

type artifactsManifestResponse struct {
	Manifest  any `json:"manifest"`
	Artifacts any `json:"artifacts"`
}

func writeReportMD(dataDir, id string, rep *doctor.Report) error {
	dir := filepath.Join(dataDir, "doctor")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	b, _ := json.MarshalIndent(rep, "", "  ")
	path := filepath.Join(dir, id+".json")
	return os.WriteFile(path, b, 0o644)
}
