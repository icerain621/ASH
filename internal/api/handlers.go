package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/ash-repwiki/ash/internal/config"
	"github.com/ash-repwiki/ash/internal/doctor"
	"github.com/ash-repwiki/ash/internal/events"
	"github.com/ash-repwiki/ash/internal/memory"
	"github.com/ash-repwiki/ash/internal/rules"
	"github.com/ash-repwiki/ash/internal/runs"
	"github.com/ash-repwiki/ash/internal/store"
	"github.com/ash-repwiki/ash/internal/toolbus"
)

type Handler struct {
	db            *store.DB
	runs          *runs.Service
	events        *events.Service
	scenarios     *rules.Loader
	doctor        *doctor.Service
	doctorReports *reportStore
	memory        *memory.Service
}

func NewHandler(db *store.DB, scenarios *rules.Loader) *Handler {
	ev := events.NewService(db)
	tools := toolbus.DefaultBus()
	runsSvc := runs.NewService(db, ev, scenarios, tools)
	return &Handler{
		db:        db,
		events:    ev,
		scenarios: scenarios,
		runs:      runsSvc,
		doctor:    doctor.NewService(runsSvc, ev, scenarios, db.DataDir()),
		memory:    memory.NewService(db, ev),
	}
}

func (h *Handler) Register(r *gin.Engine, webDir string) {
	cfg := config.Load()
	r.Use(corsMiddleware())
	r.GET("/healthz", h.healthz)
	r.GET("/readyz", h.readyz)
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))
	registerSwagger(r)

	v1 := r.Group("/api/v1")
	v1.Use(authMiddleware(cfg))
	{
		v1.POST("/runs", h.createRun)
		v1.GET("/runs", h.listRuns)
		v1.GET("/runs/:runId", h.getRun)
		v1.GET("/runs/:runId/stream", h.streamRun)
		v1.GET("/runs/:runId/artifacts", h.listRunArtifacts)
		v1.GET("/runs/:runId/artifacts/:artifactName/access", h.getRunArtifactAccess)
		v1.GET("/runs/:runId/checkpoints", h.listRunCheckpoints)
		v1.GET("/runs/:runId/checkpoints/:checkpointId/access", h.getRunCheckpointAccess)
		v1.GET("/runs/:runId/timeline", h.getRunTimeline)
		v1.GET("/runs/:runId/tool-calls", h.listRunToolCalls)
		v1.GET("/runs/:runId/agent-tasks", h.listRunAgentTasks)
		v1.GET("/runs/:runId/quality-metrics", h.listRunQualityMetrics)
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
		v1.POST("/memory/hit-used", h.memoryHitUsed)

		v1.POST("/rag/index", h.indexRAG)
		v1.POST("/rag/query", h.queryRAG)

		v1.GET("/model-router/providers", h.listModelProviders)
		v1.POST("/model-router/route", h.routeModel)
		v1.GET("/observability/waterfall/:runId", h.getWaterfall)
		v1.GET("/observability/quality/:runId", h.getQualityMetrics)
		v1.GET("/mcp/tools", h.listMCPTools)
		v1.POST("/mcp/tools", h.registerMCPTool)
		v1.POST("/feedback", h.createFeedback)
		v1.GET("/secrets", h.listSecrets)
		v1.POST("/secrets", h.createSecret)
		v1.POST("/secrets/:secretId/rotate", h.rotateSecret)
		v1.DELETE("/secrets/:secretId", h.deleteSecret)

		v1.POST("/auth/login", h.login)
		v1.POST("/auth/dev-login", h.devLogin)
		v1.GET("/auth/me", h.authMe)
		v1.POST("/auth/password", h.changePassword)
		v1.GET("/orgs", h.listOrgs)
		v1.POST("/orgs", h.createOrg)
		v1.GET("/orgs/:orgId/roles", h.listRoles)
		v1.POST("/orgs/:orgId/roles", h.createRole)
		v1.GET("/spaces", h.listSpaces)
		v1.POST("/spaces", h.createSpace)
		v1.GET("/spaces/:spaceId/members", h.listSpaceMembers)
		v1.POST("/spaces/:spaceId/members", h.createSpaceMember)
		v1.GET("/audit/logs", h.listAuditLogs)
		v1.GET("/audit/policy", h.getAuditPolicy)
		v1.PUT("/audit/policy", h.updateAuditPolicy)
		v1.POST("/audit/retention/apply", h.applyAuditRetention)
		v1.POST("/audit/export", h.createAuditExport)
		v1.GET("/audit/exports", h.listAuditExports)
		v1.GET("/audit/exports/:exportId/access", h.getAuditExportAccess)
		v1.GET("/plugins/abi", h.getPluginABIProfile)
		v1.GET("/plugins", h.listPlugins)
		v1.POST("/plugins", h.registerPlugin)
		v1.POST("/plugins/:pluginId/verify", h.verifyPlugin)
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
		c.JSON(http.StatusServiceUnavailable, HealthResponse{Status: "not_ready", Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, HealthResponse{Status: "ready"})
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
	}
	if !h.requirePermission(c, permRunCreate, req.SpaceID) {
		return
	}
	resp, err := h.runs.Create(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("RUN_CREATE_FAILED", err.Error()))
		return
	}
	c.Header("X-Run-Id", resp.RunID)
	c.Header("X-Trace-Id", resp.TraceID)
	c.JSON(http.StatusCreated, resp)
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
	items, err := h.runs.ListForSpace(currentSpace(c), limit)
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
	sum, err := h.runs.Get(runID)
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
	if lastID := c.GetHeader("Last-Event-ID"); lastID != "" {
		if seq, err := h.events.SeqFromEventID(runID, lastID); err == nil {
			lastSeq = seq
		}
	}

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
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

	evs, err := h.events.ListAfter(runID, lastSeq, 500)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("EVENT_LIST_FAILED", err.Error()))
		return
	}
	writeEvents(evs)
	if len(evs) > 0 {
		lastSeq = evs[len(evs)-1].Seq
	}

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-ticker.C:
			evs, err := h.events.ListAfter(runID, lastSeq, 100)
			if err != nil || len(evs) == 0 {
				continue
			}
			writeEvents(evs)
			lastSeq = evs[len(evs)-1].Seq
		}
	}
}
