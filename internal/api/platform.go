package api

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ash-repwiki/ash/internal/artifactstore"
	"github.com/ash-repwiki/ash/internal/config"
	"github.com/ash-repwiki/ash/internal/modelrouter"
	"github.com/ash-repwiki/ash/internal/observability"
	"github.com/ash-repwiki/ash/internal/pluginabi"
	"github.com/ash-repwiki/ash/internal/store"
)

type registerMCPToolRequest struct {
	Name    string         `json:"name" binding:"required"`
	Server  string         `json:"server" binding:"required"`
	Schema  map[string]any `json:"schema,omitempty"`
	Risk    string         `json:"risk,omitempty"`
	SpaceID string         `json:"spaceId,omitempty"`
}

type createFeedbackRequest struct {
	TargetType string `json:"targetType" binding:"required"`
	TargetID   string `json:"targetId" binding:"required"`
	Rating     int    `json:"rating,omitempty"`
	Comment    string `json:"comment,omitempty"`
	ActorID    string `json:"actorId,omitempty"`
	SpaceID    string `json:"spaceId,omitempty"`
}

type createOrgRequest struct {
	Name string `json:"name" binding:"required"`
	Slug string `json:"slug,omitempty"`
}

type createSpaceRequest struct {
	OrgID string `json:"orgId" binding:"required"`
	Name  string `json:"name" binding:"required"`
	Slug  string `json:"slug,omitempty"`
}

type createRoleRequest struct {
	Name        string   `json:"name" binding:"required"`
	Permissions []string `json:"permissions,omitempty"`
}

type createMemberRequest struct {
	UserID      string `json:"userId,omitempty"`
	Email       string `json:"email,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
	Password    string `json:"password,omitempty"`
	RoleID      string `json:"roleId" binding:"required"`
	Status      string `json:"status,omitempty"`
}

type updateAuditPolicyRequest struct {
	RetentionDays int  `json:"retentionDays"`
	RedactPayload bool `json:"redactPayload"`
}

type applyAuditRetentionRequest struct {
	DryRun bool `json:"dryRun,omitempty"`
}

type registerPluginRequest struct {
	Name         string   `json:"name" binding:"required"`
	Version      string   `json:"version" binding:"required"`
	Protocol     string   `json:"protocol,omitempty"`
	ABI          string   `json:"abi,omitempty"`
	Endpoint     string   `json:"endpoint" binding:"required"`
	Capabilities []string `json:"capabilities,omitempty"`
	SpaceID      string   `json:"spaceId,omitempty"`
}

// ListModelProviders godoc
// @Summary List configured model providers
// @Tags model-router
// @Produce json
// @Success 200 {object} ModelProviderListResponse
// @Router /api/v1/model-router/providers [get]
func (h *Handler) listModelProviders(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"items": modelrouter.NewFromEnv().Providers()})
}

// RouteModel godoc
// @Summary Route a non-coding model request
// @Tags model-router
// @Accept json
// @Produce json
// @Param body body modelrouter.Request true "routing request"
// @Success 200 {object} modelrouter.Decision
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/model-router/route [post]
func (h *Handler) routeModel(c *gin.Context) {
	var req modelrouter.Request
	_ = c.ShouldBindJSON(&req)
	spaceID := currentSpace(c)
	if req.RunID != "" {
		if !h.requireRunAccess(c, req.RunID) {
			return
		}
		var err error
		spaceID, err = h.runSpaceID(req.RunID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errorBody("RUN_SCOPE_CHECK_FAILED", err.Error()))
			return
		}
	}
	if !h.requirePermission(c, permModelRoute, spaceID) {
		return
	}
	decision := modelrouter.NewFromEnv().Route(req)
	if req.RunID != "" || req.StepID != "" {
		row := modelrouter.UsageRow(decision, req)
		if err := h.db.Create(&row).Error; err != nil {
			c.JSON(http.StatusInternalServerError, errorBody("MODEL_USAGE_RECORD_FAILED", err.Error()))
			return
		}
	}
	c.JSON(http.StatusOK, decision)
}

// GetWaterfall godoc
// @Summary Get run waterfall spans
// @Tags observability
// @Produce json
// @Param runId path string true "run id"
// @Success 200 {object} observability.Waterfall
// @Failure 404 {object} APIErrorResponse
// @Router /api/v1/observability/waterfall/{runId} [get]
func (h *Handler) getWaterfall(c *gin.Context) {
	runID := c.Param("runId")
	if !h.requireRunAccess(c, runID) {
		return
	}
	waterfall, err := observability.BuildWaterfall(h.db, runID)
	if err != nil {
		c.JSON(http.StatusNotFound, errorBody("WATERFALL_NOT_FOUND", err.Error()))
		return
	}
	c.JSON(http.StatusOK, waterfall)
}

// GetQualityMetrics godoc
// @Summary Get run quality metrics
// @Tags observability
// @Produce json
// @Param runId path string true "run id"
// @Success 200 {object} QualityMetricListResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/observability/quality/{runId} [get]
func (h *Handler) getQualityMetrics(c *gin.Context) {
	runID := c.Param("runId")
	if !h.requireRunAccess(c, runID) {
		return
	}
	items, err := h.runs.QualityMetrics(runID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("QUALITY_METRIC_LIST_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, QualityMetricListResponse{Items: items})
}

// ListMCPTools godoc
// @Summary List registered MCP tools
// @Tags mcp
// @Produce json
// @Success 200 {object} MCPToolListResponse
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/mcp/tools [get]
func (h *Handler) listMCPTools(c *gin.Context) {
	var rows []store.MCPTool
	if err := h.db.Where("space_id = ?", currentSpace(c)).Order("created_at desc").Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("MCP_TOOL_LIST_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": rows})
}

// RegisterMCPTool godoc
// @Summary Register an MCP tool
// @Tags mcp
// @Accept json
// @Produce json
// @Param body body registerMCPToolRequest true "MCP tool"
// @Success 201 {object} store.MCPTool
// @Failure 400 {object} APIErrorResponse
// @Failure 403 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/mcp/tools [post]
func (h *Handler) registerMCPTool(c *gin.Context) {
	var req registerMCPToolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", err.Error()))
		return
	}
	space := firstNonEmptyAPI(req.SpaceID, currentSpace(c))
	if !h.requirePermission(c, permMCPWrite, space) {
		return
	}
	schema := "{}"
	if req.Schema != nil {
		b, _ := json.Marshal(req.Schema)
		schema = string(b)
	}
	risk := req.Risk
	if risk == "" {
		risk = "medium"
	}
	now := time.Now().UTC()
	row := store.MCPTool{
		ID: "mcp_" + uuid.NewString(), SpaceID: space, Name: req.Name, Server: req.Server,
		SchemaJSON: schema, Risk: risk, Status: "registered", CreatedAt: now, UpdatedAt: now,
	}
	if err := h.db.Create(&row).Error; err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("MCP_TOOL_CREATE_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusCreated, row)
}

// CreateFeedback godoc
// @Summary Create feedback
// @Tags feedback
// @Accept json
// @Produce json
// @Param body body createFeedbackRequest true "feedback"
// @Success 201 {object} store.Feedback
// @Failure 400 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/feedback [post]
func (h *Handler) createFeedback(c *gin.Context) {
	var req createFeedbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", err.Error()))
		return
	}
	space := firstNonEmptyAPI(req.SpaceID, currentSpace(c))
	if !h.requirePermission(c, permFeedbackWrite, space) {
		return
	}
	row := store.Feedback{
		ID: "fb_" + uuid.NewString(), SpaceID: space,
		TargetType: req.TargetType, TargetID: req.TargetID, Rating: req.Rating,
		Comment: req.Comment, ActorID: firstNonEmptyAPI(req.ActorID, currentActor(c)), CreatedAt: time.Now().UTC(),
	}
	if err := h.db.Create(&row).Error; err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("FEEDBACK_CREATE_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusCreated, row)
}

type devLoginRequest struct {
	SpaceID string `json:"spaceId,omitempty"`
}

func (h *Handler) devLogin(c *gin.Context) {
	var req devLoginRequest
	_ = c.ShouldBindJSON(&req)
	spaceID := firstNonEmptyAPI(req.SpaceID, "local")
	spaceName := "Local"
	if spaceID != "local" {
		var space store.Space
		if err := h.db.First(&space, "id = ?", spaceID).Error; err == nil {
			spaceName = space.Name
		}
	}
	cfg := config.Load()
	token, err := signToken(tokenClaims{
		Sub: "dev-user", SpaceID: spaceID, Role: "admin",
		Exp: time.Now().Add(24 * time.Hour).Unix(),
	}, cfg.JWTSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("TOKEN_SIGN_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user":  gin.H{"id": "dev-user", "displayName": "Dev User"},
		"space": gin.H{"id": spaceID, "name": spaceName},
	})
}

func (h *Handler) listOrgs(c *gin.Context) {
	var rows []store.Org
	if err := h.db.Order("created_at desc").Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("ORG_LIST_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": rows})
}

// CreateOrg godoc
// @Summary Create organization
// @Tags orgs
// @Accept json
// @Produce json
// @Param body body createOrgRequest true "organization"
// @Success 201 {object} store.Org
// @Failure 400 {object} APIErrorResponse
// @Failure 403 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/orgs [post]
func (h *Handler) createOrg(c *gin.Context) {
	if !h.requirePermission(c, permOrgWrite) {
		return
	}
	var req createOrgRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", err.Error()))
		return
	}
	now := time.Now().UTC()
	org := store.Org{
		ID: "org_" + uuid.NewString(), Name: strings.TrimSpace(req.Name),
		Slug: firstNonEmptyAPI(req.Slug, slugify(req.Name)), CreatedAt: now, UpdatedAt: now,
	}
	if org.Name == "" {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", "name is required"))
		return
	}
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&org).Error; err != nil {
			return err
		}
		actor := firstNonEmptyAPI(currentActor(c), "dev-user")
		user := store.User{ID: actor, DisplayName: actor, Status: "active", CreatedAt: now, UpdatedAt: now}
		if err := tx.FirstOrCreate(&user, "id = ?", actor).Error; err != nil {
			return err
		}
		role := store.Role{
			ID: "role_" + uuid.NewString(), OrgID: org.ID, Name: "admin",
			Permissions: `["*"]`, CreatedAt: now, UpdatedAt: now,
		}
		if err := tx.Create(&role).Error; err != nil {
			return err
		}
		member := store.Member{
			ID: "mem_" + uuid.NewString(), OrgID: org.ID, UserID: actor, RoleID: role.ID,
			Status: "active", CreatedAt: now, UpdatedAt: now,
		}
		if err := tx.Create(&member).Error; err != nil {
			return err
		}
		return tx.Create(auditRow(currentSpace(c), currentActor(c), "org.created", map[string]any{
			"orgId": org.ID, "name": org.Name, "slug": org.Slug,
		})).Error
	}); err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("ORG_CREATE_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusCreated, org)
}

func (h *Handler) listSpaces(c *gin.Context) {
	var rows []store.Space
	if err := h.db.Order("created_at desc").Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("SPACE_LIST_FAILED", err.Error()))
		return
	}
	if len(rows) == 0 && currentSpace(c) == "local" {
		c.JSON(http.StatusOK, gin.H{"items": []gin.H{{"id": "local", "name": "Local", "slug": "local"}}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": rows})
}

// CreateSpace godoc
// @Summary Create space
// @Tags spaces
// @Accept json
// @Produce json
// @Param body body createSpaceRequest true "space"
// @Success 201 {object} store.Space
// @Failure 400 {object} APIErrorResponse
// @Failure 403 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/spaces [post]
func (h *Handler) createSpace(c *gin.Context) {
	var req createSpaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", err.Error()))
		return
	}
	var org store.Org
	if err := h.db.First(&org, "id = ?", req.OrgID).Error; err != nil {
		c.JSON(http.StatusBadRequest, errorBody("ORG_NOT_FOUND", "org not found"))
		return
	}
	if !h.requireOrgPermission(c, org.ID, permSpaceWrite) {
		return
	}
	now := time.Now().UTC()
	space := store.Space{
		ID: "space_" + uuid.NewString(), OrgID: org.ID, Name: strings.TrimSpace(req.Name),
		Slug: firstNonEmptyAPI(req.Slug, slugify(req.Name)), CreatedAt: now, UpdatedAt: now,
	}
	if space.Name == "" {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", "name is required"))
		return
	}
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&space).Error; err != nil {
			return err
		}
		if err := tx.Create(&store.ResourceScope{
			ID: "scope_" + uuid.NewString(), SpaceID: space.ID,
			ResourceType: "space", ResourceID: space.ID, PolicyJSON: `{"inheritsOrg":true}`,
			CreatedAt: now, UpdatedAt: now,
		}).Error; err != nil {
			return err
		}
		return tx.Create(auditRow(space.ID, currentActor(c), "space.created", map[string]any{
			"orgId": org.ID, "spaceId": space.ID, "name": space.Name, "slug": space.Slug,
		})).Error
	}); err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("SPACE_CREATE_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusCreated, space)
}

// ListRoles godoc
// @Summary List organization roles
// @Tags orgs
// @Produce json
// @Param orgId path string true "organization id"
// @Success 200 {object} RoleListResponse
// @Failure 403 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/orgs/{orgId}/roles [get]
func (h *Handler) listRoles(c *gin.Context) {
	orgID := strings.TrimSpace(c.Param("orgId"))
	var org store.Org
	if err := h.db.First(&org, "id = ?", orgID).Error; err != nil {
		c.JSON(http.StatusNotFound, errorBody("ORG_NOT_FOUND", "org not found"))
		return
	}
	if !h.requireOrgPermission(c, orgID, permRoleRead) {
		return
	}
	var rows []store.Role
	if err := h.db.Where("org_id = ?", orgID).Order("created_at desc").Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("ROLE_LIST_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, RoleListResponse{Items: rows})
}

// CreateRole godoc
// @Summary Create organization role
// @Tags orgs
// @Accept json
// @Produce json
// @Param orgId path string true "organization id"
// @Param body body createRoleRequest true "role"
// @Success 201 {object} store.Role
// @Failure 400 {object} APIErrorResponse
// @Failure 403 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/orgs/{orgId}/roles [post]
func (h *Handler) createRole(c *gin.Context) {
	orgID := strings.TrimSpace(c.Param("orgId"))
	var org store.Org
	if err := h.db.First(&org, "id = ?", orgID).Error; err != nil {
		c.JSON(http.StatusNotFound, errorBody("ORG_NOT_FOUND", "org not found"))
		return
	}
	if !h.requireOrgPermission(c, orgID, permRoleWrite) {
		return
	}
	var req createRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", err.Error()))
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", "name is required"))
		return
	}
	perms := normalizePermissions(req.Permissions)
	rawPerms, _ := json.Marshal(perms)
	now := time.Now().UTC()
	row := store.Role{
		ID: "role_" + uuid.NewString(), OrgID: orgID, Name: name,
		Permissions: string(rawPerms), CreatedAt: now, UpdatedAt: now,
	}
	if err := h.db.Create(&row).Error; err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("ROLE_CREATE_FAILED", err.Error()))
		return
	}
	_ = h.db.Create(auditRow(currentSpace(c), currentActor(c), "role.created", map[string]any{
		"orgId": orgID, "roleId": row.ID, "name": row.Name, "permissions": perms,
	})).Error
	c.JSON(http.StatusCreated, row)
}

// ListSpaceMembers godoc
// @Summary List space members
// @Tags spaces
// @Produce json
// @Param spaceId path string true "space id"
// @Success 200 {object} MemberListResponse
// @Failure 403 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/spaces/{spaceId}/members [get]
func (h *Handler) listSpaceMembers(c *gin.Context) {
	space, ok := h.spaceForParam(c)
	if !ok {
		return
	}
	if !h.requirePermission(c, permMemberRead, space.ID) {
		return
	}
	var rows []store.Member
	if err := h.db.Where("space_id = ?", space.ID).Order("created_at desc").Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("MEMBER_LIST_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, MemberListResponse{Items: rows})
}

// CreateSpaceMember godoc
// @Summary Add a member to a space
// @Tags spaces
// @Accept json
// @Produce json
// @Param spaceId path string true "space id"
// @Param body body createMemberRequest true "member"
// @Success 201 {object} store.Member
// @Failure 400 {object} APIErrorResponse
// @Failure 403 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/spaces/{spaceId}/members [post]
func (h *Handler) createSpaceMember(c *gin.Context) {
	space, ok := h.spaceForParam(c)
	if !ok {
		return
	}
	if !h.requirePermission(c, permMemberWrite, space.ID) {
		return
	}
	var req createMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", err.Error()))
		return
	}
	userID := strings.TrimSpace(req.UserID)
	email := strings.TrimSpace(req.Email)
	displayName := strings.TrimSpace(req.DisplayName)
	if userID == "" {
		userID = email
	}
	if userID == "" {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", "userId or email is required"))
		return
	}
	var role store.Role
	if err := h.db.First(&role, "id = ?", req.RoleID).Error; err != nil {
		c.JSON(http.StatusBadRequest, errorBody("ROLE_NOT_FOUND", "role not found"))
		return
	}
	if role.OrgID != "" && role.OrgID != space.OrgID {
		c.JSON(http.StatusBadRequest, errorBody("ROLE_SCOPE_MISMATCH", "role does not belong to the space organization"))
		return
	}
	status := firstNonEmptyAPI(strings.TrimSpace(req.Status), "active")
	passwordHash := ""
	if strings.TrimSpace(req.Password) != "" {
		if len(req.Password) < 8 {
			c.JSON(http.StatusBadRequest, errorBody("WEAK_PASSWORD", "password must be at least 8 characters"))
			return
		}
		var err error
		passwordHash, err = hashPassword(req.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errorBody("PASSWORD_HASH_FAILED", err.Error()))
			return
		}
	}
	now := time.Now().UTC()
	var member store.Member
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		user := store.User{
			ID: userID, Email: email, DisplayName: firstNonEmptyAPI(displayName, userID),
			PasswordHash: passwordHash, Status: "active", CreatedAt: now, UpdatedAt: now,
		}
		if err := tx.FirstOrCreate(&user, "id = ?", userID).Error; err != nil {
			return err
		}
		updates := map[string]any{"updated_at": now}
		if email != "" {
			updates["email"] = email
		}
		if displayName != "" {
			updates["display_name"] = displayName
		}
		if passwordHash != "" {
			updates["password_hash"] = passwordHash
		}
		if err := tx.Model(&store.User{}).Where("id = ?", userID).Updates(updates).Error; err != nil {
			return err
		}
		member = store.Member{
			ID: "mem_" + uuid.NewString(), OrgID: space.OrgID, SpaceID: space.ID,
			UserID: userID, RoleID: role.ID, Status: status, CreatedAt: now, UpdatedAt: now,
		}
		if err := tx.Create(&member).Error; err != nil {
			return err
		}
		return tx.Create(auditRow(space.ID, currentActor(c), "member.added", map[string]any{
			"orgId": space.OrgID, "spaceId": space.ID, "memberId": member.ID,
			"userId": userID, "roleId": role.ID, "status": status,
		})).Error
	}); err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("MEMBER_CREATE_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusCreated, member)
}

func (h *Handler) createAuditExport(c *gin.Context) {
	var req struct {
		SpaceID     string `json:"spaceId,omitempty"`
		RequestedBy string `json:"requestedBy,omitempty"`
	}
	_ = c.ShouldBindJSON(&req)
	space := firstNonEmptyAPI(req.SpaceID, currentSpace(c))
	if !h.requirePermission(c, permAuditExport, space) {
		return
	}
	now := time.Now().UTC()
	row := store.AuditExport{
		ID: "audexp_" + uuid.NewString(), SpaceID: space,
		Status: "running", RequestedBy: firstNonEmptyAPI(req.RequestedBy, currentActor(c)), CreatedAt: now,
	}
	if err := h.db.Create(&row).Error; err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("AUDIT_EXPORT_CREATE_FAILED", err.Error()))
		return
	}
	var logs []store.AuditLog
	if err := h.db.Where("space_id = ?", space).Order("created_at asc").Find(&logs).Error; err != nil {
		row.Status = "failed"
		_ = h.db.Save(&row).Error
		c.JSON(http.StatusInternalServerError, errorBody("AUDIT_EXPORT_FAILED", err.Error()))
		return
	}
	payload := map[string]any{"exportId": row.ID, "spaceId": space, "createdAt": now.UnixMilli(), "logs": logs}
	b, _ := json.MarshalIndent(payload, "", "  ")
	b = append(b, '\n')
	digest := sha256Digest(b)
	cfg := config.Load()
	artifactStore := artifactstore.New(cfg.ArtifactStore, h.db.DataDir())
	storeKey := "audit_exports/" + space + "/" + row.ID + ".json"
	ref, err := artifactStore.Put(context.Background(), storeKey, bytes.NewReader(b), "application/json")
	if err != nil {
		row.Status = "failed"
		_ = h.db.Save(&row).Error
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
	_ = h.db.Save(&row).Error
	_ = h.db.Create(auditRow(space, currentActor(c), "audit.export_completed", map[string]any{
		"exportId": row.ID, "uri": row.URI, "digest": row.Digest, "sizeBytes": row.SizeBytes,
	})).Error
	c.JSON(http.StatusAccepted, row)
}

// ListAuditExports godoc
// @Summary List audit exports
// @Tags audit
// @Produce json
// @Success 200 {object} AuditExportListResponse
// @Failure 403 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/audit/exports [get]
func (h *Handler) listAuditExports(c *gin.Context) {
	space := currentSpace(c)
	if !h.requirePermission(c, permAuditExport, space) {
		return
	}
	var rows []store.AuditExport
	if err := h.db.Where("space_id = ?", space).Order("created_at desc").Limit(100).Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("AUDIT_EXPORT_LIST_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, AuditExportListResponse{Items: rows})
}

// GetAuditExportAccess godoc
// @Summary Get signed audit export access
// @Tags audit
// @Produce json
// @Param exportId path string true "audit export id"
// @Param ttlSeconds query int false "ttl seconds" default(900)
// @Success 200 {object} AuditExportAccessResponse
// @Failure 403 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/audit/exports/{exportId}/access [get]
func (h *Handler) getAuditExportAccess(c *gin.Context) {
	space := currentSpace(c)
	if !h.requirePermission(c, permAuditExport, space) {
		return
	}
	var row store.AuditExport
	if err := h.db.First(&row, "id = ? AND space_id = ?", c.Param("exportId"), space).Error; err != nil {
		c.JSON(http.StatusNotFound, errorBody("AUDIT_EXPORT_NOT_FOUND", "audit export not found"))
		return
	}
	if row.Status != "completed" || row.StoreKey == "" {
		c.JSON(http.StatusNotFound, errorBody("AUDIT_EXPORT_NOT_READY", "audit export is not ready"))
		return
	}
	ttlSeconds, _ := strconv.Atoi(c.DefaultQuery("ttlSeconds", "900"))
	if ttlSeconds <= 0 || ttlSeconds > 3600 {
		ttlSeconds = 900
	}
	ttl := time.Duration(ttlSeconds) * time.Second
	signed, err := artifactstore.New(config.Load().ArtifactStore, h.db.DataDir()).SignedURL(context.Background(), row.StoreKey, ttl)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("AUDIT_EXPORT_ACCESS_FAILED", err.Error()))
		return
	}
	expires := time.Now().UTC().Add(ttl)
	_ = h.db.Create(auditRow(space, currentActor(c), "audit.export_access_issued", map[string]any{
		"exportId": row.ID, "digest": row.Digest, "ttlSeconds": ttlSeconds,
	})).Error
	c.JSON(http.StatusOK, AuditExportAccessResponse{
		ExportID: row.ID, URI: row.URI, SignedURL: signed, ExpiresAt: expires.UnixMilli(),
		Digest: row.Digest, ContentType: row.ContentType, SizeBytes: row.SizeBytes,
	})
}

// ListAuditLogs godoc
// @Summary List audit logs
// @Tags audit
// @Produce json
// @Param eventType query string false "event type filter"
// @Param runId query string false "run id filter"
// @Param q query string false "event or payload search"
// @Param limit query int false "max items" default(100)
// @Success 200 {object} AuditLogListResponse
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/audit/logs [get]
func (h *Handler) listAuditLogs(c *gin.Context) {
	space := currentSpace(c)
	if !h.requirePermission(c, permAuditExport, space) {
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	q := h.db.Where("space_id = ?", space)
	if eventType := strings.TrimSpace(c.Query("eventType")); eventType != "" {
		q = q.Where("event_type = ?", eventType)
	}
	if runID := strings.TrimSpace(c.Query("runId")); runID != "" {
		q = q.Where("run_id = ?", runID)
	}
	if text := strings.TrimSpace(c.Query("q")); text != "" {
		like := "%" + strings.ToLower(text) + "%"
		q = q.Where("LOWER(event_type) LIKE ? OR LOWER(payload_json) LIKE ?", like, like)
	}
	var rows []store.AuditLog
	if err := q.Order("created_at desc").Limit(limit).Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("AUDIT_LOG_LIST_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, AuditLogListResponse{Items: rows})
}

// GetAuditPolicy godoc
// @Summary Get audit retention policy
// @Tags audit
// @Produce json
// @Success 200 {object} store.AuditPolicy
// @Failure 403 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/audit/policy [get]
func (h *Handler) getAuditPolicy(c *gin.Context) {
	space := currentSpace(c)
	if !h.requirePermission(c, permAuditExport, space) {
		return
	}
	policy, err := h.auditPolicy(space)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("AUDIT_POLICY_GET_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, policy)
}

// UpdateAuditPolicy godoc
// @Summary Update audit retention policy
// @Tags audit
// @Accept json
// @Produce json
// @Param body body updateAuditPolicyRequest true "audit policy"
// @Success 200 {object} store.AuditPolicy
// @Failure 400 {object} APIErrorResponse
// @Failure 403 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/audit/policy [put]
func (h *Handler) updateAuditPolicy(c *gin.Context) {
	space := currentSpace(c)
	if !h.requirePermission(c, permAuditExport, space) {
		return
	}
	var req updateAuditPolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", err.Error()))
		return
	}
	if req.RetentionDays <= 0 || req.RetentionDays > 3650 {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_AUDIT_POLICY", "retentionDays must be 1..3650"))
		return
	}
	policy, err := h.auditPolicy(space)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("AUDIT_POLICY_GET_FAILED", err.Error()))
		return
	}
	if policy.Locked {
		c.JSON(http.StatusForbidden, errorBody("AUDIT_POLICY_LOCKED", "audit policy is locked"))
		return
	}
	policy.RetentionDays = req.RetentionDays
	policy.RedactPayload = req.RedactPayload
	policy.UpdatedAt = time.Now().UTC()
	if err := h.db.Save(policy).Error; err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("AUDIT_POLICY_UPDATE_FAILED", err.Error()))
		return
	}
	_ = h.db.Create(auditRow(space, currentActor(c), "audit.policy_updated", map[string]any{
		"retentionDays": policy.RetentionDays, "redactPayload": policy.RedactPayload,
	})).Error
	c.JSON(http.StatusOK, policy)
}

// ApplyAuditRetention godoc
// @Summary Apply audit retention policy
// @Tags audit
// @Accept json
// @Produce json
// @Param body body applyAuditRetentionRequest true "retention request"
// @Success 200 {object} AuditRetentionApplyResponse
// @Failure 403 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/audit/retention/apply [post]
func (h *Handler) applyAuditRetention(c *gin.Context) {
	space := currentSpace(c)
	if !h.requirePermission(c, permAuditExport, space) {
		return
	}
	var req applyAuditRetentionRequest
	_ = c.ShouldBindJSON(&req)
	policy, err := h.auditPolicy(space)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("AUDIT_POLICY_GET_FAILED", err.Error()))
		return
	}
	cutoff := time.Now().UTC().AddDate(0, 0, -policy.RetentionDays)
	q := h.db.Model(&store.AuditLog{}).Where("space_id = ? AND created_at < ?", space, cutoff)
	var matched int64
	if err := q.Count(&matched).Error; err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("AUDIT_RETENTION_COUNT_FAILED", err.Error()))
		return
	}
	deleted := int64(0)
	if !req.DryRun && matched > 0 {
		res := h.db.Where("space_id = ? AND created_at < ?", space, cutoff).Delete(&store.AuditLog{})
		if res.Error != nil {
			c.JSON(http.StatusInternalServerError, errorBody("AUDIT_RETENTION_APPLY_FAILED", res.Error.Error()))
			return
		}
		deleted = res.RowsAffected
	}
	resp := AuditRetentionApplyResponse{
		SpaceID: space, RetentionDays: policy.RetentionDays, Cutoff: cutoff,
		Matched: matched, Deleted: deleted, DryRun: req.DryRun,
	}
	_ = h.db.Create(auditRow(space, currentActor(c), "audit.retention_applied", resp)).Error
	c.JSON(http.StatusOK, resp)
}

// ListPlugins godoc
// @Summary List registered plugins
// @Tags plugins
// @Produce json
// @Success 200 {object} PluginRegistryListResponse
// @Failure 403 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/plugins [get]
func (h *Handler) listPlugins(c *gin.Context) {
	if !h.requirePermission(c, permPluginRead, currentSpace(c)) {
		return
	}
	var rows []store.PluginRegistry
	if err := h.db.Where("space_id = ?", currentSpace(c)).Order("created_at desc").Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("PLUGIN_LIST_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, PluginRegistryListResponse{Items: rows})
}

// RegisterPlugin godoc
// @Summary Register a plugin
// @Tags plugins
// @Accept json
// @Produce json
// @Param body body registerPluginRequest true "plugin registration"
// @Success 201 {object} store.PluginRegistry
// @Failure 400 {object} APIErrorResponse
// @Failure 403 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/plugins [post]
func (h *Handler) registerPlugin(c *gin.Context) {
	var req registerPluginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", err.Error()))
		return
	}
	space := firstNonEmptyAPI(req.SpaceID, currentSpace(c))
	if !h.requirePermission(c, permPluginWrite, space) {
		return
	}
	protocol := firstNonEmptyAPI(strings.ToLower(strings.TrimSpace(req.Protocol)), "grpc")
	abi := normalizePluginABI(req.ABI)
	caps, _ := json.Marshal(req.Capabilities)
	compatible, lastErr := pluginCompatibility(protocol, abi, req.Name, req.Version, req.Endpoint)
	status := "registered"
	if !compatible {
		status = "incompatible"
	}
	now := time.Now().UTC()
	row := store.PluginRegistry{
		ID: "plg_" + uuid.NewString(), SpaceID: space,
		Name: strings.TrimSpace(req.Name), Version: strings.TrimSpace(req.Version),
		Protocol: protocol, ABI: abi, Endpoint: strings.TrimSpace(req.Endpoint),
		Capabilities: string(caps), Compatible: compatible, Status: status, LastError: lastErr,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := h.db.Create(&row).Error; err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("PLUGIN_REGISTER_FAILED", err.Error()))
		return
	}
	_ = h.db.Create(auditRow(space, currentActor(c), "plugin.registered", map[string]any{
		"pluginId": row.ID, "name": row.Name, "version": row.Version,
		"protocol": row.Protocol, "abi": row.ABI, "compatible": row.Compatible,
	})).Error
	c.JSON(http.StatusCreated, row)
}

// VerifyPlugin godoc
// @Summary Verify plugin ABI compatibility
// @Tags plugins
// @Produce json
// @Param pluginId path string true "plugin id"
// @Success 200 {object} store.PluginRegistry
// @Failure 403 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/plugins/{pluginId}/verify [post]
func (h *Handler) verifyPlugin(c *gin.Context) {
	var row store.PluginRegistry
	if err := h.db.First(&row, "id = ? AND space_id = ?", c.Param("pluginId"), currentSpace(c)).Error; err != nil {
		c.JSON(http.StatusNotFound, errorBody("PLUGIN_NOT_FOUND", "plugin not found"))
		return
	}
	if !h.requirePermission(c, permPluginWrite, row.SpaceID) {
		return
	}
	compatible, lastErr := pluginCompatibility(row.Protocol, row.ABI, row.Name, row.Version, row.Endpoint)
	row.Compatible = compatible
	row.LastError = lastErr
	row.Status = "verified"
	if !compatible {
		row.Status = "incompatible"
	}
	row.UpdatedAt = time.Now().UTC()
	if err := h.db.Save(&row).Error; err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("PLUGIN_VERIFY_FAILED", err.Error()))
		return
	}
	_ = h.db.Create(auditRow(row.SpaceID, currentActor(c), "plugin.verified", map[string]any{
		"pluginId": row.ID, "protocol": row.Protocol, "abi": row.ABI,
		"compatible": row.Compatible, "lastError": row.LastError,
	})).Error
	c.JSON(http.StatusOK, row)
}

func (h *Handler) auditPolicy(space string) (*store.AuditPolicy, error) {
	space = firstNonEmptyAPI(space, "local")
	var policy store.AuditPolicy
	if err := h.db.First(&policy, "space_id = ?", space).Error; err == nil {
		return &policy, nil
	} else if err != gorm.ErrRecordNotFound {
		return nil, err
	}
	now := time.Now().UTC()
	policy = store.AuditPolicy{
		SpaceID: space, RetentionDays: 365, RedactPayload: false,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := h.db.Create(&policy).Error; err != nil {
		return nil, err
	}
	return &policy, nil
}

func pluginCompatibility(protocol, abi, name, version, endpoint string) (bool, string) {
	if ok, reason := pluginabi.Compatible(protocol, abi, name, version); !ok {
		return false, reason
	}
	if strings.TrimSpace(endpoint) == "" {
		return false, "endpoint is required"
	}
	return true, ""
}

func (h *Handler) spaceForParam(c *gin.Context) (store.Space, bool) {
	spaceID := strings.TrimSpace(c.Param("spaceId"))
	var space store.Space
	if err := h.db.First(&space, "id = ?", spaceID).Error; err != nil {
		c.JSON(http.StatusNotFound, errorBody("SPACE_NOT_FOUND", "space not found"))
		return store.Space{}, false
	}
	return space, true
}

func normalizePermissions(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	return out
}

func firstNonEmptyAPI(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func slugify(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "_", "-")
	fields := strings.Fields(value)
	if len(fields) == 0 {
		return "default"
	}
	return strings.Join(fields, "-")
}

func auditRow(spaceID, actorID, eventType string, payload any) *store.AuditLog {
	b, _ := json.Marshal(payload)
	return &store.AuditLog{
		ID: "aud_" + uuid.NewString(), SpaceID: firstNonEmptyAPI(spaceID, "local"),
		ActorID: actorID, EventType: eventType, PayloadJSON: string(b), CreatedAt: time.Now().UTC(),
	}
}

func sha256Digest(b []byte) string {
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:])
}
