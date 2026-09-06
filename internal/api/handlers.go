package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ash-repwiki/ash/internal/alerts"
	"github.com/ash-repwiki/ash/internal/ci"
	"github.com/ash-repwiki/ash/internal/config"
	"github.com/ash-repwiki/ash/internal/diffreview"
	"github.com/ash-repwiki/ash/internal/doctor"
	"github.com/ash-repwiki/ash/internal/events"
	"github.com/ash-repwiki/ash/internal/evolve"
	"github.com/ash-repwiki/ash/internal/goal"
	"github.com/ash-repwiki/ash/internal/harness"
	"github.com/ash-repwiki/ash/internal/improve"
	"github.com/ash-repwiki/ash/internal/knowledge"
	"github.com/ash-repwiki/ash/internal/memory"
	metricssvc "github.com/ash-repwiki/ash/internal/metrics"
	"github.com/ash-repwiki/ash/internal/opsenv"
	"github.com/ash-repwiki/ash/internal/quest"
	"github.com/ash-repwiki/ash/internal/releases"
	"github.com/ash-repwiki/ash/internal/rules"
	"github.com/ash-repwiki/ash/internal/runs"
	"github.com/ash-repwiki/ash/internal/scenariopatch"
	"github.com/ash-repwiki/ash/internal/secrets"
	"github.com/ash-repwiki/ash/internal/session"
	"github.com/ash-repwiki/ash/internal/spacerules"
	"github.com/ash-repwiki/ash/internal/store"
	"github.com/ash-repwiki/ash/internal/toolbus"
	"github.com/ash-repwiki/ash/internal/waker"
)

type Handler struct {
	db            *store.DB
	runs          *runs.Service
	events        *events.Service
	scenarios     *rules.Loader
	doctor        *doctor.Service
	doctorReports *reportStore
	memory        *memory.Service
	improve       *improve.Service
	ci            *ci.Service
	metrics       *metricssvc.Service
	alerts        *alerts.Service
	releases      *releases.Service
	harness       *harness.Service
	patches       *scenariopatch.Service
	evolve        *evolve.Service
	goal          *goal.Service
	quest         *quest.Service
	diffReview    *diffreview.Service
	knowledge     *knowledge.Service
	spaceRules    *spacerules.Service
	session       *session.Service
	waker         *waker.Service
}

func NewHandler(db *store.DB, scenarios *rules.Loader) *Handler {
	ev := events.NewService(db)
	tools := toolbus.DefaultBus()
	runsSvc := runs.NewService(db, ev, scenarios, tools)
	ciSvc := ci.ApplyFixtureProvider(ci.NewService(db, func(spaceID, secretID string) (string, error) {
		var row store.SecretRecord
		if err := db.First(&row, "id = ? AND space_id = ? AND status = ?", secretID, spaceID, "active").Error; err != nil {
			return "", err
		}
		return secrets.Open(row.ValueCiphertext, config.Load().SecretKey)
	}))
	memSvc := memory.NewService(db, ev)
	harSvc := harness.NewService(db)
	patchSvc := scenariopatch.NewService(db)
	improveSvc := improve.NewService(db, runsSvc, ev)
	runsSvc.SetImproveDrafter(improveSvc)
	goalSvc := goal.NewService(db, scenarios, runsSvc, ev)
	sessionSvc := session.NewService(db, goalSvc, ev)
	runsSvc.WithSessionService(sessionRunLinker{svc: sessionSvc})
	h := &Handler{
		db:         db,
		events:     ev,
		scenarios:  scenarios,
		runs:       runsSvc,
		doctor:     doctor.NewService(runsSvc, ev, scenarios, db.DataDir()),
		memory:     memSvc,
		improve:    improveSvc,
		ci:         ciSvc,
		metrics:    metricssvc.NewService(db),
		alerts:     alerts.NewService(db),
		releases:   releases.NewService(db),
		harness:    harSvc,
		patches:    patchSvc,
		evolve:     evolve.NewService(db, memSvc, harSvc, patchSvc),
		goal:       goalSvc,
		quest:      quest.NewService(db, runsSvc),
		diffReview: diffreview.NewService(db, runsSvc),
		knowledge:  knowledge.NewService(db, memSvc),
		spaceRules: spacerules.NewService(db),
		session:    sessionSvc,
		waker:      waker.NewService(db),
	}
	if h.doctor != nil {
		adapter := doctorWakerAdapter{svc: h.doctor}
		h.waker = h.waker.WithDoctorRunner(adapter)
		waker.SetDefaultDoctorRunner(adapter)
	}
	return h
}

func (h *Handler) Register(r *gin.Engine, webDir string) {
	cfg := config.Load()
	r.Use(corsMiddleware())
	r.GET("/healthz", h.healthz)
	r.GET("/readyz", h.readyz)
	r.GET("/metrics", h.prometheusMetrics)
	registerSwagger(r)

	v1 := r.Group("/api/v1")
	v1.Use(authMiddleware(cfg))
	v1.Use(h.rlsMiddleware())
	{
		v1.POST("/runs", h.createRun)
		v1.POST("/runs/from-goal", h.fromGoal)
		v1.GET("/runs/plans/:planId", h.getGoalPlan)
		v1.POST("/runs/plans/:planId/approve", h.approveGoalPlan)
		v1.POST("/runs/plans/:planId/reject", h.rejectGoalPlan)
		v1.GET("/quest/board", h.questBoard)
		v1.GET("/repos/profile", h.getRepoProfile)
		v1.GET("/wiki/pages", h.listWikiPages)
		v1.GET("/wiki/pages/:pageId", h.getWikiPage)
		v1.GET("/skills", h.listSkills)
		v1.GET("/skills/catalog", h.listSkillCatalog)
		v1.POST("/skills/catalog/install", h.installSkillFromCatalog)
		v1.POST("/skills/packs/verify", h.verifySkillPack)
		v1.POST("/skills/packs/install", h.installSkillPack)
		v1.GET("/skills/:skillId", h.getSkill)
		v1.GET("/providers/agent", h.getAgentProviderStatus)
		v1.POST("/agents/sessions", h.createAgentSession)
		v1.GET("/agents/sessions/:sessionId", h.getAgentSession)
		v1.POST("/agents/sessions/:sessionId/turns", h.promptAgentSessionTurn)
		v1.GET("/agents/sessions/:sessionId/events", h.listAgentSessionEvents)
		v1.GET("/runs", h.listRuns)
		v1.GET("/runs/:runId", h.getRun)
		v1.GET("/runs/:runId/tree", h.getRunTree)
		v1.POST("/runs/:runId/sub-runs", h.spawnSubRun)
		v1.GET("/runs/:runId/stream", h.streamRun)
		v1.GET("/runs/:runId/artifacts", h.listRunArtifacts)
		v1.GET("/runs/:runId/artifacts/:artifactName/access", h.getRunArtifactAccess)
		v1.GET("/runs/:runId/checkpoints", h.listRunCheckpoints)
		v1.GET("/runs/:runId/checkpoints/:checkpointId/access", h.getRunCheckpointAccess)
		v1.GET("/runs/:runId/timeline", h.getRunTimeline)
		v1.GET("/runs/:runId/diff", h.getRunDiff)
		v1.GET("/runs/:runId/diff/comments", h.listDiffComments)
		v1.POST("/runs/:runId/diff/comments", h.createDiffComment)
		v1.POST("/runs/:runId/steps/:stepId/rate", h.rateRunStep)
		v1.GET("/runs/:runId/tool-calls", h.listRunToolCalls)
		v1.GET("/runs/:runId/agent-tasks", h.listRunAgentTasks)
		v1.GET("/runs/:runId/quality-metrics", h.listRunQualityMetrics)
		v1.GET("/runs/:runId/provenance", h.getRunProvenance)
		v1.POST("/runs/:runId/resume", h.resumeRun)
		v1.POST("/runs/:runId/replay", h.replayRun)
		v1.POST("/runs/:runId/cancel", h.cancelRun)
		v1.POST("/runs/:runId/approve", h.approveRun)
		v1.GET("/approvals", h.listApprovals)
		v1.POST("/approvals/:approvalId/approve", h.approveApproval)
		v1.POST("/approvals/:approvalId/reject", h.rejectApproval)

		v1.POST("/doctor/run", h.runDoctor)
		v1.GET("/doctor/reports/:reportId", h.getDoctorReport)

		v1.GET("/scenarios", h.listScenarios)
		v1.GET("/scenarios/:name/:version", h.getScenario)
		v1.POST("/scenarios/validate", h.validateScenario)
		v1.POST("/scenarios/validate/raw", h.validateScenarioRaw)

		v1.POST("/memory/candidates", h.createMemoryCandidate)
		v1.GET("/memory/candidates", h.listMemoryCandidates)
		v1.POST("/memory/candidates/:candidateId/review", h.reviewMemoryCandidate)
		v1.GET("/memory/records/:recordId", h.getMemoryRecord)
		v1.POST("/memory/query", h.queryMemory)
		v1.POST("/memory/migrate", h.runMemoryMigration)
		v1.GET("/memory/ttl-queue", h.getMemoryTTLQueue)
		v1.POST("/memory/ttl-sweep", h.sweepMemoryTTL)
		v1.POST("/memory/hit-used", h.memoryHitUsed)

		v1.GET("/waker/queue", h.getWakerQueue)
		v1.POST("/waker/sweep", h.postWakerSweep)
		v1.GET("/waker/status", h.getWakerStatus)
		v1.GET("/waker/duties", h.getWakerDuties)
		v1.POST("/waker/duties/:id/run", h.postWakerDutyRun)
		v1.POST("/waker/duties/:id/enable", h.postWakerDutyEnable)

		v1.POST("/improve/proposals", h.createImproveProposal)
		v1.GET("/improve/proposals", h.listImproveProposals)
		v1.GET("/improve/proposals/:proposalId", h.getImproveProposal)
		v1.POST("/improve/proposals/:proposalId/experiment", h.startImproveExperiment)
		v1.POST("/improve/proposals/:proposalId/canary", h.startImproveCanary)
		v1.POST("/improve/proposals/:proposalId/promote", h.promoteImproveProposal)
		v1.POST("/improve/proposals/:proposalId/rollback", h.rollbackImproveProposal)

		v1.POST("/rag/index", h.indexRAG)
		v1.POST("/rag/symbols/rebuild", h.rebuildRAGSymbols)
		v1.POST("/rag/lsp/hover", h.ragLSPHover)
		v1.POST("/rag/lsp/definition", h.ragLSPDefinition)
		v1.POST("/rag/lsp/references", h.ragLSPReferences)
		v1.POST("/rag/query", h.queryRAG)
		v1.GET("/rag/profile", h.getRAGProfile)

		v1.GET("/model-router/providers", h.listModelProviders)
		v1.POST("/model-router/route", h.routeModel)
		v1.GET("/observability/waterfall/:runId", h.getWaterfall)
		v1.GET("/observability/otel/status", h.getOtelStatus)
		v1.GET("/observability/quality/:runId", h.getQualityMetrics)
		v1.GET("/observability/alerts", h.listAlerts)
		v1.GET("/observability/alert-rules", h.listAlertRules)
		v1.PUT("/observability/alert-rules", h.putAlertRules)
		v1.POST("/observability/alerts/evaluate", h.evaluateAlerts)
		v1.GET("/observability/trace/:traceId", h.getTrace)
		v1.GET("/tools/risk-catalog", h.listToolRiskCatalog)
		v1.GET("/mcp/tools", h.listMCPTools)
		v1.POST("/mcp/tools", h.registerMCPTool)
		v1.POST("/feedback", h.createFeedback)
		v1.GET("/feedback", h.listFeedback)
		v1.PATCH("/feedback/:feedbackId", h.updateFeedback)

		v1.GET("/reviews/queue", h.listReviewsQueue)
		v1.POST("/reviews/:reviewId/decide", h.decideReview)
		v1.POST("/repo/connections", h.createRepoConnection)
		v1.GET("/repo/connections", h.listRepoConnections)
		v1.GET("/ci/runs", h.listCIRuns)
		v1.GET("/ci/jobs", h.listCIJobs)
		v1.POST("/ci/failures/diagnose", h.diagnoseCIFailure)
		v1.GET("/ci/diagnoses", h.listCIDiagnoses)
		v1.POST("/ci/diagnoses/:diagnosisId/adopt", h.adoptCIDiagnosis)
		v1.POST("/ci/diagnoses/:diagnosisId/dismiss", h.dismissCIDiagnosis)
		v1.POST("/webhooks/github", h.githubWebhook)
		v1.GET("/metrics/overview", h.getMetricsOverview)
		v1.GET("/metrics/prometheus", h.getMetricsPrometheus)
		v1.GET("/secrets", h.listSecrets)
		v1.POST("/secrets", h.createSecret)
		v1.POST("/secrets/:secretId/rotate", h.rotateSecret)
		v1.DELETE("/secrets/:secretId", h.deleteSecret)

		v1.GET("/harness/profiles", h.listHarnessProfiles)
		v1.POST("/harness/profiles", h.createHarnessProfile)
		v1.GET("/harness/profiles/active", h.loadActiveHarnessProfile)
		v1.GET("/harness/profiles/:profileId", h.getHarnessProfile)
		v1.PUT("/harness/profiles/:profileId", h.updateHarnessProfile)
		v1.POST("/harness/profiles/:profileId/submit-review", h.submitHarnessProfileReview)
		v1.POST("/harness/profiles/:profileId/promote", h.promoteHarnessProfile)
		v1.POST("/harness/profiles/:profileId/rollback", h.rollbackHarnessProfile)

		v1.GET("/scenario-patches", h.listScenarioPatches)
		v1.POST("/scenario-patches", h.createScenarioPatch)
		v1.POST("/scenario-patches/:patchId/submit-review", h.submitScenarioPatchReview)

		v1.POST("/releases", h.createRelease)
		v1.GET("/releases", h.listReleases)
		v1.GET("/releases/:releaseId/checklist", h.getReleaseChecklist)
		v1.PATCH("/releases/:releaseId/checklist", h.patchReleaseChecklist)
		v1.POST("/releases/:releaseId/gate", h.evaluateReleaseGate)
		v1.POST("/releases/:releaseId/rollback-drills", h.createRollbackDrill)

		v1.POST("/auth/login", h.login)
		v1.POST("/auth/dev-login", h.devLogin)
		v1.GET("/auth/me", h.authMe)
		v1.POST("/auth/password", h.changePassword)
		v1.GET("/orgs", h.listOrgs)
		v1.POST("/orgs", h.createOrg)
		v1.GET("/org-templates", h.listOrgTemplates)
		v1.POST("/org-templates/:templateId/provision", h.provisionOrgTemplate)
		v1.GET("/orgs/:orgId/roles", h.listRoles)
		v1.POST("/orgs/:orgId/roles", h.createRole)
		v1.GET("/spaces", h.listSpaces)
		v1.POST("/spaces", h.createSpace)
		v1.GET("/spaces/:spaceId/members", h.listSpaceMembers)
		v1.POST("/spaces/:spaceId/members", h.createSpaceMember)
		v1.GET("/spaces/:spaceId/resource-scopes", h.listSpaceResourceScopes)
		v1.PUT("/spaces/:spaceId/resource-scopes/:scopeId", h.updateSpaceResourceScope)
		v1.GET("/spaces/:spaceId/rules", h.getSpaceRules)
		v1.PUT("/spaces/:spaceId/rules", h.putSpaceRules)
		v1.POST("/spaces/:spaceId/rules/import", h.importSpaceRules)
		v1.POST("/spaces/:spaceId/rules/export", h.exportSpaceRules)
		v1.POST("/spaces/:spaceId/rules/preview", h.previewSpaceRules)

		v1.GET("/compliance/secret-scan", h.complianceSecretScan)
		v1.POST("/compliance/export", h.complianceExportBundle)
		v1.GET("/data-policy", h.getDataPolicy)
		v1.POST("/events/retention/apply", h.applyEventsRetention)
		v1.POST("/artifacts/retention/apply", h.applyArtifactsRetention)

		v1.GET("/scale/readiness", h.scaleReadiness)
		v1.GET("/permissions/matrix", h.permissionMatrix)
		v1.GET("/spaces/:spaceId/permissions/matrix", h.spacePermissionMatrix)
		v1.GET("/audit/logs", h.listAuditLogs)
		v1.GET("/audit/policy", h.getAuditPolicy)
		v1.PUT("/audit/policy", h.updateAuditPolicy)
		v1.POST("/audit/retention/apply", h.applyAuditRetention)
		v1.POST("/audit/export", h.createAuditExport)
		v1.GET("/audit/exports", h.listAuditExports)
		v1.GET("/audit/exports/:exportId/access", h.getAuditExportAccess)
		v1.GET("/plugins/abi", h.getPluginABIProfile)
		v1.GET("/plugins/health", h.getPluginHealth)
		v1.GET("/plugins", h.listPlugins)
		v1.POST("/plugins", h.registerPlugin)
		v1.POST("/plugins/:pluginId/verify", h.verifyPlugin)
		v1.POST("/plugins/:pluginId/export-report", h.reportPluginExport)
		v1.GET("/storage/profile", h.getStorageProfile)
	}
	registerWebUI(r, webDir)
}

func errorBody(code, message string) gin.H {
	return gin.H{"error": gin.H{"code": code, "message": message}}
}

// Healthz godoc
// @Summary Liveness probe
// @Tags health
// @Produce json
// @Success 200 {object} HealthResponse
// @Router /healthz [get]
func (h *Handler) healthz(c *gin.Context) {
	c.JSON(http.StatusOK, HealthResponse{Status: "ok"})
}

// Readyz godoc
// @Summary Readiness probe
// @Tags health
// @Produce json
// @Success 200 {object} HealthResponse
// @Failure 503 {object} HealthResponse
// @Router /readyz [get]
func (h *Handler) readyz(c *gin.Context) {
	sqlDB, err := h.db.DB.DB()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, HealthResponse{Status: "not_ready", Error: err.Error()})
		return
	}
	if err := sqlDB.Ping(); err != nil {
		c.JSON(http.StatusServiceUnavailable, h.readyzResponse("not_ready", err.Error()))
		return
	}
	c.JSON(http.StatusOK, h.readyzResponse("ready", ""))
}

func (h *Handler) readyzResponse(status, errMsg string) HealthResponse {
	ops := workerOpsSnapshot()
	profile, _ := store.DatabaseProfile(h.db.DataDir(), store.RuntimeDatabaseURL())
	if profile.PostgresRLSEnabled && h.db.Dialect() == "postgres" {
		if n, err := store.CountPostgresRLSPolicies(h.db); err == nil {
			profile.PostgresRLSPolicyCount = n
		}
	}
	migSnap, _ := store.MigrationSnapshotFor(h.db.DataDir())
	rlsEnv := store.PostgresRLSEnabled()
	resp := HealthResponse{
		Status:                    status,
		Region:                    config.Region(),
		Dialect:                   h.db.Dialect(),
		Error:                     errMsg,
		SchemaMode:                profile.SchemaMode,
		SQLMigrationVersion:       profile.SQLMigrationVersion,
		SQLMigrationExpected:      profile.SQLMigrationExpected,
		PostgresRLSEnabled:        profile.PostgresRLSEnabled,
		PostgresRLSPolicyCount:    profile.PostgresRLSPolicyCount,
		PostgresRLSPolicyExpected: profile.PostgresRLSPolicyExpected,
		RLSCatalogSummary:         rlsCatalogSummary(profile),
		ReadinessWarnings:         scaleReadinessWarnings(profile, migSnap),
		LiveGateHints:             opsenv.LiveGateHints(),
		OtelEnabled:               ops.OtelEnabled,
		AlertsEvalInterval:        ops.AlertsEvalInterval,
		MemoryTTLSweepInterval:    ops.MemoryTTLSweepInterval,
		MetricsEventReplayEnabled: ops.MetricsEventReplayEnabled,
		RetentionEventsDays:       config.EffectiveRetentionEventsDays(),
		RetentionAuditDays:        config.EffectiveRetentionAuditDays(),
		RetentionArtifactsDays:    config.EffectiveRetentionArtifactsDays(),
		RetentionArtifactsMaxRuns: config.EffectiveRetentionArtifactsMaxRuns(),
	}
	if rlsEnv && resp.PostgresRLSPolicyExpected == 0 {
		resp.PostgresRLSPolicyExpected = int64(store.PostgresRLSExpectedPolicyCount())
	}
	return resp
}

// CreateRun godoc
// @Summary Create and start a run
// @Description Validates scenario inputs, executes scenario steps, and records run events.
// @Tags runs
// @Accept json
// @Produce json
// @Param body body runs.CreateRequest true "run request"
// @Success 201 {object} runs.CreateResponse
// @Header 201 {string} X-Run-Id "created run id"
// @Header 201 {string} X-Trace-Id "trace id"
// @Failure 400 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/runs [post]
func (h *Handler) createRun(c *gin.Context) {
	var req runs.CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", err.Error()))
		return
	}
	if req.SpaceID == "" {
		req.SpaceID = currentSpace(c)
	} else if !h.requireRequestSpace(c, req.SpaceID) {
		return
	}
	if !h.requirePermission(c, permRunCreate, req.SpaceID) {
		return
	}
	if strings.TrimSpace(req.ActorRole) == "" {
		req.ActorRole = currentRole(c)
	}
	resp, err := h.runsFor(c).Create(req)
	if resp == nil {
		c.JSON(http.StatusInternalServerError, errorBody("RUN_CREATE_FAILED", err.Error()))
		return
	}
	c.Header("X-Run-Id", resp.RunID)
	c.Header("X-Trace-Id", resp.TraceID)
	out := RunCreateResponse{RunID: resp.RunID, TraceID: resp.TraceID}
	if err != nil {
		out.ExecutionError = err.Error()
		if sum, getErr := h.runsFor(c).Get(resp.RunID); getErr == nil {
			out.Status = sum.Status
		} else {
			out.Status = "failed"
		}
	}
	c.JSON(http.StatusCreated, out)
}

// ListRuns godoc
// @Summary List runs
// @Tags runs
// @Produce json
// @Param limit query int false "max items" default(50)
// @Success 200 {object} RunListResponse
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/runs [get]
func (h *Handler) listRuns(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	items, err := h.runsFor(c).ListForSpace(currentSpace(c), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("RUN_LIST_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, RunListResponse{Items: items})
}

// GetRun godoc
// @Summary Get run summary
// @Tags runs
// @Produce json
// @Param runId path string true "run id"
// @Success 200 {object} runs.Summary
// @Failure 404 {object} APIErrorResponse
// @Router /api/v1/runs/{runId} [get]
func (h *Handler) getRun(c *gin.Context) {
	runID := c.Param("runId")
	if !h.requireRunAccess(c, runID) {
		return
	}
	sum, err := h.runsFor(c).Get(runID)
	if err != nil {
		c.JSON(http.StatusNotFound, errorBody("RUN_NOT_FOUND", "run not found"))
		return
	}
	c.JSON(http.StatusOK, sum)
}

// StreamRun godoc
// @Summary Stream run events (SSE)
// @Description Server-Sent Events; supports Last-Event-ID for resume.
// @Tags runs
// @Produce text/event-stream
// @Param runId path string true "run id"
// @Param Last-Event-ID header string false "last received event id"
// @Success 200 {string} string "SSE stream"
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/runs/{runId}/stream [get]
func (h *Handler) auditStream(c *gin.Context, runID, eventType string, payload map[string]any) {
	row := auditRow(currentSpace(c), currentActor(c), eventType, payload)
	row.RunID = runID
	_ = h.dbFor(c).Create(row).Error
}

func (h *Handler) streamRun(c *gin.Context) {
	runID := c.Param("runId")
	if !h.requireRunAccess(c, runID) {
		return
	}

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")

	lastSeq := int64(0)
	lastID := strings.TrimSpace(c.GetHeader("Last-Event-ID"))
	if lastID == "" {
		lastID = strings.TrimSpace(c.Query("Last-Event-ID"))
	}
	if lastID == "" {
		lastID = strings.TrimSpace(c.Query("lastEventId"))
	}
	if lastID != "" {
		if seq, err := h.eventsFor(c).SeqFromEventID(runID, lastID); err == nil {
			lastSeq = seq
		}
	}

	opened := false
	failed := false
	defer func() {
		if opened && !failed {
			h.auditStream(c, runID, "stream.session_closed", map[string]any{
				"runId":  runID,
				"reason": "client_disconnect",
			})
		}
	}()

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		h.auditStream(c, runID, "stream.session_failed", map[string]any{
			"runId": runID, "reason": "sse_unsupported",
		})
		c.JSON(http.StatusInternalServerError, errorBody("SSE_UNSUPPORTED", "streaming not supported"))
		return
	}

	writeEvents := func(evs []events.Envelope) {
		for _, ev := range evs {
			data, _ := json.Marshal(ev)
			_, _ = c.Writer.Write([]byte("id: " + ev.ID + "\n"))
			_, _ = c.Writer.Write([]byte("event: " + ev.Type + "\n"))
			_, _ = c.Writer.Write([]byte("data: "))
			_, _ = c.Writer.Write(data)
			_, _ = c.Writer.Write([]byte("\n\n"))
			flusher.Flush()
		}
	}

	evs, err := h.eventsFor(c).ListAfter(runID, lastSeq, 500)
	if err != nil {
		h.auditStream(c, runID, "stream.session_failed", map[string]any{
			"runId": runID, "reason": "event_list_failed", "error": err.Error(),
		})
		c.JSON(http.StatusInternalServerError, errorBody("EVENT_LIST_FAILED", err.Error()))
		return
	}
	writeEvents(evs)
	if len(evs) > 0 {
		lastSeq = evs[len(evs)-1].Seq
	}
	opened = true
	h.auditStream(c, runID, "stream.session_opened", map[string]any{
		"runId": runID, "resumedFromSeq": lastSeq,
	})

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-ticker.C:
			evs, err := h.eventsFor(c).ListAfter(runID, lastSeq, 100)
			if err != nil {
				failed = true
				h.auditStream(c, runID, "stream.session_failed", map[string]any{
					"runId": runID, "reason": "event_poll_failed", "error": err.Error(),
				})
				return
			}
			if len(evs) == 0 {
				continue
			}
			writeEvents(evs)
			lastSeq = evs[len(evs)-1].Seq
		}
	}
}
