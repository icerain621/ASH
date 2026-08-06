package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ash-repwiki/ash/internal/ci"
	metricssvc "github.com/ash-repwiki/ash/internal/metrics"
	"github.com/ash-repwiki/ash/internal/store"
)

type createRepoConnectionRequest struct {
	Provider      string `json:"provider,omitempty"`
	Owner         string `json:"owner" binding:"required"`
	Repo          string `json:"repo" binding:"required"`
	DefaultBranch string `json:"defaultBranch,omitempty"`
	SecretID      string `json:"secretId" binding:"required"`
	SpaceID       string `json:"spaceId,omitempty"`
}

type diagnoseCIFailureRequest struct {
	ConnectionID string `json:"connectionId,omitempty"`
	RunID        string `json:"runId,omitempty"`
	JobID        string `json:"jobId,omitempty"`
	LogText      string `json:"logText,omitempty"`
}

type RepoConnectionListResponse struct {
	Items []store.RepoConnection `json:"items"`
}

type CIRunListResponse struct {
	Items []store.CIRun `json:"items"`
}

type CIJobListResponse struct {
	Items []store.CIJob `json:"items"`
}

type CIDiagnosisListResponse struct {
	Items []ci.DiagnosisResponse `json:"items"`
}

type decideCIDiagnosisRequest struct {
	Reason string `json:"reason,omitempty"`
}

// CreateRepoConnection godoc
// @Summary Create a GitHub repository connection
// @Tags repo
// @Accept json
// @Produce json
// @Param body body createRepoConnectionRequest true "repo connection"
// @Success 201 {object} store.RepoConnection
// @Failure 400 {object} APIErrorResponse
// @Failure 403 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/repo/connections [post]
func (h *Handler) createRepoConnection(c *gin.Context) {
	var raw map[string]any
	if err := c.ShouldBindJSON(&raw); err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", err.Error()))
		return
	}
	for _, key := range []string{"token", "githubToken", "accessToken"} {
		if _, ok := raw[key]; ok {
			c.JSON(http.StatusBadRequest, errorBody("PLAINTEXT_TOKEN_REJECTED", "repo connections must reference secretId; plaintext tokens are not accepted"))
			return
		}
	}
	var req createRepoConnectionRequest
	rawBytes, _ := json.Marshal(raw)
	if err := json.Unmarshal(rawBytes, &req); err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", err.Error()))
		return
	}
	space := firstNonEmptyAPI(req.SpaceID, currentSpace(c))
	if !h.requireTargetSpace(c, space) {
		return
	}
	if !h.requirePermission(c, permRepoWrite, space) {
		return
	}
	if err := h.requireSecretReference(c, space, req.SecretID); err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_SECRET_REFERENCE", err.Error()))
		return
	}
	row, err := h.ciFor(c).CreateConnection(ci.CreateConnectionRequest{
		SpaceID: space, Provider: req.Provider, Owner: req.Owner, Repo: req.Repo,
		DefaultBranch: req.DefaultBranch, SecretID: req.SecretID, CreatedBy: currentActor(c),
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("REPO_CONNECTION_CREATE_FAILED", err.Error()))
		return
	}
	_ = h.dbFor(c).Create(auditRow(space, currentActor(c), "repo.connection_created", map[string]any{
		"connectionId": row.ID, "provider": row.Provider, "owner": row.Owner, "repo": row.Repo,
	}))
	c.JSON(http.StatusCreated, row)
}

// ListRepoConnections godoc
// @Summary List repository connections for the current space
// @Tags repo
// @Produce json
// @Success 200 {object} RepoConnectionListResponse
// @Failure 403 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/repo/connections [get]
func (h *Handler) listRepoConnections(c *gin.Context) {
	space := currentSpace(c)
	if !h.requirePermission(c, permRepoRead, space) {
		return
	}
	rows, err := h.ciFor(c).ListConnections(space)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("REPO_CONNECTION_LIST_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, RepoConnectionListResponse{Items: rows})
}

// ListCIRuns godoc
// @Summary List CI workflow run snapshots
// @Tags ci
// @Produce json
// @Param connectionId query string false "repo connection id"
// @Param limit query int false "max items" default(50)
// @Param sync query bool false "sync from GitHub before listing"
// @Success 200 {object} CIRunListResponse
// @Failure 403 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/ci/runs [get]
func (h *Handler) listCIRuns(c *gin.Context) {
	space := currentSpace(c)
	if !h.requirePermission(c, permCIRead, space) {
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	sync := strings.EqualFold(c.Query("sync"), "true") || c.Query("sync") == "1"
	rows, err := h.ciFor(c).ListRuns(c.Request.Context(), space, c.Query("connectionId"), limit, sync)
	if err != nil {
		writeCIProviderError(c, "CI_RUN_LIST_FAILED", err)
		return
	}
	c.JSON(http.StatusOK, CIRunListResponse{Items: rows})
}

// ListCIJobs godoc
// @Summary List CI workflow jobs
// @Tags ci
// @Produce json
// @Param runId query string false "ci run id"
// @Param limit query int false "max items" default(50)
// @Param sync query bool false "sync from GitHub before listing"
// @Success 200 {object} CIJobListResponse
// @Failure 403 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/ci/jobs [get]
func (h *Handler) listCIJobs(c *gin.Context) {
	space := currentSpace(c)
	if !h.requirePermission(c, permCIRead, space) {
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	sync := strings.EqualFold(c.Query("sync"), "true") || c.Query("sync") == "1"
	rows, err := h.ciFor(c).ListJobs(c.Request.Context(), space, c.Query("runId"), limit, sync)
	if err != nil {
		writeCIProviderError(c, "CI_JOB_LIST_FAILED", err)
		return
	}
	c.JSON(http.StatusOK, CIJobListResponse{Items: rows})
}

// DiagnoseCIFailure godoc
// @Summary Diagnose a CI failure using deterministic rules
// @Tags ci
// @Accept json
// @Produce json
// @Param body body diagnoseCIFailureRequest true "CI failure target or log text"
// @Success 200 {object} ci.DiagnosisResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 403 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/ci/failures/diagnose [post]
func (h *Handler) diagnoseCIFailure(c *gin.Context) {
	space := currentSpace(c)
	if !h.requirePermission(c, permCIDiagnose, space) {
		return
	}
	var req diagnoseCIFailureRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", err.Error()))
		return
	}
	resp, err := h.ciFor(c).Diagnose(c.Request.Context(), ci.DiagnoseRequest{
		SpaceID: space, ConnectionID: req.ConnectionID, RunID: req.RunID, JobID: req.JobID, LogText: req.LogText,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("CI_DIAGNOSE_FAILED", err.Error()))
		return
	}
	_ = h.dbFor(c).Create(auditRow(space, currentActor(c), "ci.failure_diagnosed", map[string]any{
		"diagnosisId": resp.ID, "connectionId": resp.ConnectionID, "runId": resp.RunID, "jobId": resp.JobID,
		"rootCause": resp.RootCause, "confidence": resp.Confidence,
	}))
	c.JSON(http.StatusOK, resp)
}

// ListCIDiagnoses godoc
// @Summary List CI failure diagnoses
// @Tags ci
// @Produce json
// @Param connectionId query string false "repo connection id"
// @Param runId query string false "ci run id"
// @Param jobId query string false "ci job id"
// @Param decisionStatus query string false "pending|adopted|dismissed"
// @Param limit query int false "max items" default(50)
// @Success 200 {object} CIDiagnosisListResponse
// @Failure 403 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/ci/diagnoses [get]
func (h *Handler) listCIDiagnoses(c *gin.Context) {
	space := currentSpace(c)
	if !h.requirePermission(c, permCIRead, space) {
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	rows, err := h.ciFor(c).ListDiagnoses(ci.ListDiagnosesRequest{
		SpaceID: space, ConnectionID: c.Query("connectionId"), RunID: c.Query("runId"),
		JobID: c.Query("jobId"), DecisionStatus: c.Query("decisionStatus"), Limit: limit,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("CI_DIAGNOSIS_LIST_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, CIDiagnosisListResponse{Items: rows})
}

// AdoptCIDiagnosis godoc
// @Summary Mark a CI diagnosis as adopted
// @Tags ci
// @Accept json
// @Produce json
// @Param diagnosisId path string true "diagnosis id"
// @Param body body decideCIDiagnosisRequest false "decision reason"
// @Success 200 {object} ci.DiagnosisResponse
// @Failure 403 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/ci/diagnoses/{diagnosisId}/adopt [post]
func (h *Handler) adoptCIDiagnosis(c *gin.Context) {
	h.decideCIDiagnosis(c, "adopted")
}

// DismissCIDiagnosis godoc
// @Summary Mark a CI diagnosis as dismissed
// @Tags ci
// @Accept json
// @Produce json
// @Param diagnosisId path string true "diagnosis id"
// @Param body body decideCIDiagnosisRequest false "decision reason"
// @Success 200 {object} ci.DiagnosisResponse
// @Failure 403 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/ci/diagnoses/{diagnosisId}/dismiss [post]
func (h *Handler) dismissCIDiagnosis(c *gin.Context) {
	h.decideCIDiagnosis(c, "dismissed")
}

func (h *Handler) decideCIDiagnosis(c *gin.Context, decision string) {
	space := currentSpace(c)
	if !h.requirePermission(c, permCIDiagnose, space) {
		return
	}
	var req decideCIDiagnosisRequest
	_ = c.ShouldBindJSON(&req)
	resp, err := h.ciFor(c).DecideDiagnosis(ci.DecideDiagnosisRequest{
		SpaceID: space, DiagnosisID: c.Param("diagnosisId"), Decision: decision,
		Reason: req.Reason, ActorID: currentActor(c),
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("CI_DIAGNOSIS_DECISION_FAILED", err.Error()))
		return
	}
	_ = h.dbFor(c).Create(auditRow(space, currentActor(c), "ci.diagnosis_decided", map[string]any{
		"diagnosisId": resp.ID, "decision": decision, "reason": req.Reason,
		"connectionId": resp.ConnectionID, "runId": resp.RunID, "jobId": resp.JobID,
	}))
	c.JSON(http.StatusOK, resp)
}

// GetMetricsOverview godoc
// @Summary Get KPI dashboard metrics
// @Tags metrics
// @Produce json
// @Param spaceId query string false "space id"
// @Param projectId query string false "project/repo connection id"
// @Param from query string false "RFC3339 start time"
// @Param to query string false "RFC3339 end time"
// @Param period query string false "day|week"
// @Success 200 {object} metricssvc.Overview
// @Failure 400 {object} APIErrorResponse
// @Failure 403 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/metrics/overview [get]
func (h *Handler) getMetricsOverview(c *gin.Context) {
	space := firstNonEmptyAPI(c.Query("spaceId"), currentSpace(c))
	if !h.requireRequestSpace(c, space) {
		return
	}
	if !h.requirePermission(c, permCIRead, space) {
		return
	}
	from, err := parseOptionalTime(c.Query("from"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_FROM", err.Error()))
		return
	}
	to, err := parseOptionalTime(c.Query("to"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_TO", err.Error()))
		return
	}
	resp, err := h.metricsFor(c).Overview(metricssvc.OverviewRequest{
		SpaceID: space, ProjectID: c.Query("projectId"), From: from, To: to, Period: c.Query("period"),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("METRICS_OVERVIEW_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) requireSecretReference(c *gin.Context, spaceID, secretID string) error {
	secretID = strings.TrimSpace(secretID)
	if secretID == "" {
		return strconv.ErrSyntax
	}
	var row store.SecretRecord
	if err := h.dbFor(c).First(&row, "id = ? AND space_id = ? AND status = ?", secretID, spaceID, "active").Error; err != nil {
		return err
	}
	return nil
}

func parseOptionalTime(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339, value)
}

func writeCIProviderError(c *gin.Context, fallbackCode string, err error) {
	switch {
	case errors.Is(err, ci.ErrGitHubCircuitOpen), errors.Is(err, ci.ErrGitHubUnavailable):
		c.JSON(http.StatusBadGateway, errorBody("CI_PROVIDER_UNAVAILABLE", err.Error()))
	default:
		c.JSON(http.StatusInternalServerError, errorBody(fallbackCode, err.Error()))
	}
}
