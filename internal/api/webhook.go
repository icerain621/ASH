package api

import (
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ash-repwiki/ash/internal/ci"
	"github.com/ash-repwiki/ash/internal/runs"
	"github.com/ash-repwiki/ash/internal/store"
)

type githubWebhookResponse struct {
	Duplicate      bool                  `json:"duplicate"`
	Ignored        bool                  `json:"ignored,omitempty"`
	Reason         string                `json:"reason,omitempty"`
	ConnectionID   string                `json:"connectionId,omitempty"`
	SpaceID        string                `json:"spaceId,omitempty"`
	CIRunID        string                `json:"ciRunId,omitempty"`
	CIJobID        string                `json:"ciJobId,omitempty"`
	ProviderRunID  string                `json:"providerRunId,omitempty"`
	Conclusion     string                `json:"conclusion,omitempty"`
	Workflow       string                `json:"workflow,omitempty"`
	ShouldStartRun bool                  `json:"shouldStartRun,omitempty"`
	Diagnosis      *ci.DiagnosisResponse `json:"diagnosis,omitempty"`
	AshRunID       string                `json:"ashRunId,omitempty"`
	AshTraceID     string                `json:"ashTraceId,omitempty"`
	ExecutionError string                `json:"executionError,omitempty"`
}

// GitHubWebhook godoc
// @Summary Ingest GitHub Actions webhook (HMAC)
// @Description Public path: verifies X-Hub-Signature-256, upserts CI run/job, diagnoses failures, optionally creates a hotfix Run when autoRun=1.
// @Tags webhooks
// @Accept json
// @Produce json
// @Param connectionId query string true "repo connection id"
// @Param autoRun query bool false "create hotfix run on failure"
// @Param repoRoot query string false "repo root for autoRun" default(.)
// @Param X-Hub-Signature-256 header string true "sha256 HMAC of raw body"
// @Param X-GitHub-Delivery header string false "delivery id for idempotency"
// @Param X-GitHub-Event header string false "event name" default(workflow_run)
// @Param body body object true "GitHub workflow_run payload"
// @Success 200 {object} githubWebhookResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 401 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/webhooks/github [post]
func (h *Handler) githubWebhook(c *gin.Context) {
	connectionID := strings.TrimSpace(c.Query("connectionId"))
	if connectionID == "" {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", "connectionId query is required"))
		return
	}
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", "unable to read body"))
		return
	}
	// Lookup connection under RLS bypass (public path has no JWT space yet).
	bypassCtx := store.WithRLSBypassContext(c.Request.Context())
	svc := h.ci.WithContext(bypassCtx)
	conn, err := svc.ConnectionByID(bypassCtx, connectionID)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("REPO_CONNECTION_NOT_FOUND", err.Error()))
		return
	}
	secret, err := svc.ResolveWebhookSecret(conn.SpaceID, conn.SecretID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, errorBody("WEBHOOK_SECRET_UNRESOLVED", err.Error()))
		return
	}
	sig := c.GetHeader("X-Hub-Signature-256")
	if !ci.VerifyGitHubSignature(secret, body, sig) {
		c.JSON(http.StatusUnauthorized, errorBody("WEBHOOK_SIGNATURE_INVALID", "invalid X-Hub-Signature-256"))
		return
	}
	setIdentity(c, "webhook:github", conn.SpaceID, "admin")
	ctx := c.Request.Context()
	if store.PostgresRLSEnabled() {
		ctx = store.WithRLSSpaceContext(ctx, conn.SpaceID)
	}
	c.Request = c.Request.WithContext(ctx)

	autoRun := strings.EqualFold(c.Query("autoRun"), "true") || c.Query("autoRun") == "1"
	repoRoot := firstNonEmptyAPI(strings.TrimSpace(c.Query("repoRoot")), ".")
	result, err := h.ci.WithContext(ctx).IngestGitHubWebhook(ctx, ci.WebhookIngestRequest{
		ConnectionID: connectionID,
		DeliveryID:   c.GetHeader("X-GitHub-Delivery"),
		EventName:    firstNonEmptyAPI(c.GetHeader("X-GitHub-Event"), "workflow_run"),
		Body:         body,
		AutoRun:      autoRun,
		RepoRoot:     repoRoot,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("WEBHOOK_INGEST_FAILED", err.Error()))
		return
	}

	resp := githubWebhookResponse{
		Duplicate: result.Duplicate, Ignored: result.Ignored, Reason: result.Reason,
		ConnectionID: result.ConnectionID, SpaceID: result.SpaceID,
		CIRunID: result.CIRunID, CIJobID: result.CIJobID, ProviderRunID: result.ProviderRunID,
		Conclusion: result.Conclusion, Workflow: result.Workflow,
		ShouldStartRun: result.ShouldStartRun, Diagnosis: result.Diagnosis,
	}
	if result.ShouldStartRun && result.Diagnosis != nil && !result.Duplicate {
		issue := ci.IssueOrSpecFromDiagnosis(result.Workflow, result.Conclusion, "", *result.Diagnosis)
		create, createErr := h.runsFor(c).Create(runs.CreateRequest{
			Scenario:      runs.ScenarioRef{Name: "hotfix", ScenarioVersion: "1.1.0"},
			PolicyProfile: "hotfix",
			SpaceID:       conn.SpaceID,
			ActorRole:     "maintainer",
			Inputs: map[string]any{
				"issueOrSpec": issue,
				"repoRoot":    repoRoot,
			},
		})
		if create != nil {
			resp.AshRunID = create.RunID
			resp.AshTraceID = create.TraceID
			c.Header("X-Run-Id", create.RunID)
			c.Header("X-Trace-Id", create.TraceID)
		}
		if createErr != nil {
			resp.ExecutionError = createErr.Error()
		}
		_ = h.dbFor(c).Create(auditRow(conn.SpaceID, "webhook:github", "ci.webhook_autorun", map[string]any{
			"connectionId": conn.ID, "ciRunId": result.CIRunID, "diagnosisId": result.Diagnosis.ID,
			"ashRunId": resp.AshRunID, "executionError": resp.ExecutionError,
		})).Error
	}
	c.JSON(http.StatusOK, resp)
}
