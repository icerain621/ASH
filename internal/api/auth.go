package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"github.com/ash-repwiki/ash/internal/authz"
	"github.com/ash-repwiki/ash/internal/config"
	"github.com/ash-repwiki/ash/internal/store"
)

const (
	ctxActorID = "actorId"
	ctxSpaceID = "spaceId"
	ctxRole    = "role"
)

const (
	permAuditExport        = "audit:export"
	permArtifactRead       = "artifact:read"
	permCIRead             = "ci:read"
	permCIDiagnose         = "ci:diagnose"
	permFeedbackRead       = "feedback:read"
	permFeedbackWrite      = "feedback:write"
	permMemberRead         = "member:read"
	permMemberWrite        = "member:write"
	permMemoryCreate       = "memory:create"
	permMemoryQuery        = "memory:query"
	permMemoryRead         = "memory:read"
	permMemoryReview       = "memory:review"
	permMemoryUse          = "memory:use"
	permMCPWrite           = "mcp:write"
	permModelRoute         = "model:route"
	permObservabilityRead  = "observability:read"
	permObservabilityWrite = "observability:write"
	permOrgWrite           = "org:write"
	permPluginRead         = "plugin:read"
	permPluginWrite        = "plugin:write"
	permRAGIndex           = "rag:index"
	permRAGQuery           = "rag:query"
	permRepoRead           = "repo:read"
	permRepoWrite          = "repo:write"
	permReleaseRead        = "release:read"
	permReleaseWrite       = "release:write"
	permRoleRead           = "role:read"
	permRoleWrite          = "role:write"
	permRunApprove         = "run:approve"
	permRunCancel          = "run:cancel"
	permRunCreate          = "run:create"
	permSecretRead         = "secret:read"
	permSecretWrite        = "secret:write"
	permSpaceWrite         = "space:write"
	permStorageRead        = "storage:read"
)

type tokenClaims struct {
	Sub     string `json:"sub"`
	SpaceID string `json:"spaceId"`
	Role    string `json:"role"`
	Exp     int64  `json:"exp"`
}

type loginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
	SpaceID  string `json:"spaceId,omitempty"`
}

type changePasswordRequest struct {
	CurrentPassword string `json:"currentPassword" binding:"required"`
	NewPassword     string `json:"newPassword" binding:"required"`
}

// Login godoc
// @Summary Login with local credentials
// @Tags auth
// @Accept json
// @Produce json
// @Param body body loginRequest true "login credentials"
// @Success 200 {object} AuthSessionResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 401 {object} APIErrorResponse
// @Failure 403 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/auth/login [post]
func (h *Handler) login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", err.Error()))
		return
	}
	user, err := h.userByLogin(c, req.Email)
	if err != nil || user.PasswordHash == "" || !checkPasswordHash(req.Password, user.PasswordHash) {
		c.JSON(http.StatusUnauthorized, errorBody("INVALID_CREDENTIALS", "invalid email or password"))
		return
	}
	if user.Status != "" && user.Status != "active" {
		c.JSON(http.StatusForbidden, errorBody("USER_DISABLED", "user is not active"))
		return
	}
	spaceID := strings.TrimSpace(req.SpaceID)
	if spaceID == "" {
		spaceID, err = h.defaultLoginSpace(c, user.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errorBody("LOGIN_SCOPE_FAILED", err.Error()))
			return
		}
	}
	spaceID = firstNonEmptyAPI(spaceID, "local")
	if spaceID != "local" {
		ok, err := h.userHasSpaceAccess(c, user.ID, spaceID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errorBody("LOGIN_SCOPE_FAILED", err.Error()))
			return
		}
		if !ok {
			c.JSON(http.StatusForbidden, errorBody("SPACE_ACCESS_DENIED", "user is not a member of the requested space"))
			return
		}
	}
	cfg := config.Load()
	token, err := signToken(tokenClaims{
		Sub: user.ID, SpaceID: spaceID, Role: "viewer",
		Exp: time.Now().Add(24 * time.Hour).Unix(),
	}, cfg.JWTSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("TOKEN_SIGN_FAILED", err.Error()))
		return
	}
	spaceName := "Local"
	if spaceID != "local" {
		var space store.Space
		if err := h.dbBypass(c).First(&space, "id = ?", spaceID).Error; err == nil {
			spaceName = space.Name
		}
	}
	_ = h.dbBypass(c).Create(auditRow(spaceID, user.ID, "auth.login", map[string]any{
		"userId": user.ID, "email": user.Email, "spaceId": spaceID,
	})).Error
	c.JSON(http.StatusOK, AuthSessionResponse{
		Token: token,
		User:  AuthUser{ID: user.ID, Email: user.Email, DisplayName: user.DisplayName},
		Space: AuthSpace{ID: spaceID, Name: spaceName},
	})
}

// ChangePassword godoc
// @Summary Change current user's password
// @Tags auth
// @Accept json
// @Produce json
// @Param body body changePasswordRequest true "password change"
// @Success 200 {object} PasswordChangeResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 401 {object} APIErrorResponse
// @Failure 403 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/auth/password [post]
func (h *Handler) changePassword(c *gin.Context) {
	actorID := currentActor(c)
	if actorID == "" {
		c.JSON(http.StatusUnauthorized, errorBody("UNAUTHORIZED", "missing authenticated actor"))
		return
	}
	var req changePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", err.Error()))
		return
	}
	if len(req.NewPassword) < 8 {
		c.JSON(http.StatusBadRequest, errorBody("WEAK_PASSWORD", "newPassword must be at least 8 characters"))
		return
	}
	var user store.User
	if err := h.db.First(&user, "id = ?", actorID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, errorBody("USER_NOT_FOUND", "authenticated user not found"))
		return
	}
	if user.Status != "" && user.Status != "active" {
		c.JSON(http.StatusForbidden, errorBody("USER_DISABLED", "user is not active"))
		return
	}
	if user.PasswordHash == "" || !checkPasswordHash(req.CurrentPassword, user.PasswordHash) {
		c.JSON(http.StatusUnauthorized, errorBody("INVALID_CREDENTIALS", "invalid current password"))
		return
	}
	nextHash, err := hashPassword(req.NewPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("PASSWORD_HASH_FAILED", err.Error()))
		return
	}
	now := time.Now().UTC()
	if err := h.db.Model(&store.User{}).Where("id = ?", user.ID).Updates(map[string]any{
		"password_hash": nextHash,
		"updated_at":    now,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("PASSWORD_UPDATE_FAILED", err.Error()))
		return
	}
	_ = h.dbFor(c).Create(auditRow(currentSpace(c), user.ID, "auth.password_changed", map[string]any{
		"userId": user.ID,
	})).Error
	c.JSON(http.StatusOK, PasswordChangeResponse{OK: true})
}

// AuthMe godoc
// @Summary Get current authenticated identity
// @Tags auth
// @Produce json
// @Success 200 {object} AuthMeResponse
// @Failure 401 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/auth/me [get]
func (h *Handler) authMe(c *gin.Context) {
	actorID := currentActor(c)
	if actorID == "" {
		c.JSON(http.StatusUnauthorized, errorBody("UNAUTHORIZED", "missing authenticated actor"))
		return
	}
	spaceID := currentSpace(c)
	role := firstNonEmptyAPI(currentRole(c), "viewer")
	user := AuthUser{ID: actorID}
	var row store.User
	if err := h.db.First(&row, "id = ?", actorID).Error; err == nil {
		user.Email = row.Email
		user.DisplayName = row.DisplayName
	} else if actorID == "dev-user" {
		user.DisplayName = "Dev User"
	} else {
		c.JSON(http.StatusUnauthorized, errorBody("USER_NOT_FOUND", "authenticated user not found"))
		return
	}
	space := AuthSpace{ID: spaceID, Name: "Local"}
	if spaceID != "local" {
		var spaceRow store.Space
		if err := h.dbFor(c).First(&spaceRow, "id = ?", spaceID).Error; err != nil {
			c.JSON(http.StatusInternalServerError, errorBody("SPACE_LOOKUP_FAILED", err.Error()))
			return
		}
		space.Name = spaceRow.Name
	}
	perms := []string{}
	if roleAllowsPermission(role, "*") || role == "admin" {
		perms = append(perms, "*")
	} else {
		memberPerms, err := h.memberPermissions(c, actorID, spaceID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errorBody("PERMISSION_LOOKUP_FAILED", err.Error()))
			return
		}
		perms = append(perms, memberPerms...)
	}
	c.JSON(http.StatusOK, AuthMeResponse{
		User:        user,
		Space:       space,
		Role:        role,
		Permissions: normalizePermissions(perms),
	})
}

func authMiddleware(cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		if cfg.AuthMode == "disabled" || isPublicAuthPath(c.Request.URL.Path) {
			c.Next()
			return
		}
		token := bearerToken(c.GetHeader("Authorization"))
		if token == "" && cfg.AuthMode == "dev" {
			setIdentity(c, "dev-user", devSpaceOverride(c), "admin")
			c.Next()
			return
		}
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, errorBody("UNAUTHORIZED", "missing bearer token"))
			return
		}
		claims, err := verifyToken(token, cfg.JWTSecret)
		if err != nil {
			if cfg.AuthMode == "dev" && token == "dev-token" {
				setIdentity(c, "dev-user", devSpaceOverride(c), "admin")
				c.Next()
				return
			}
			c.AbortWithStatusJSON(http.StatusUnauthorized, errorBody("UNAUTHORIZED", err.Error()))
			return
		}
		setIdentity(c, claims.Sub, firstNonEmptyAPI(claims.SpaceID, "local"), firstNonEmptyAPI(claims.Role, "viewer"))
		c.Next()
	}
}

func isPublicAuthPath(path string) bool {
	switch path {
	case "/api/v1/auth/login", "/api/v1/auth/dev-login":
		return true
	default:
		return false
	}
}

func (h *Handler) userByLogin(c *gin.Context, login string) (store.User, error) {
	login = strings.TrimSpace(login)
	var user store.User
	err := h.dbBypass(c).Where("LOWER(email) = ? OR id = ?", strings.ToLower(login), login).Take(&user).Error
	return user, err
}

func (h *Handler) defaultLoginSpace(c *gin.Context, userID string) (string, error) {
	var member store.Member
	err := h.dbBypass(c).Where("user_id = ? AND status = ? AND space_id <> ''", userID, "active").
		Order("created_at asc").
		Take(&member).Error
	if err == nil && member.SpaceID != "" {
		return member.SpaceID, nil
	}
	return "local", nil
}

func (h *Handler) userHasSpaceAccess(c *gin.Context, userID, spaceID string) (bool, error) {
	if spaceID != "local" {
		var space store.Space
		if err := h.dbBypass(c).First(&space, "id = ?", spaceID).Error; err != nil {
			return false, err
		}
	}
	targetOrgID, err := h.spaceOrgID(c, spaceID)
	if err != nil {
		return false, err
	}
	var rows []memberPermissionRow
	err = h.dbBypass(c).Table("members").
		Select("members.org_id, members.space_id, roles.permissions").
		Joins("JOIN roles ON roles.id = members.role_id").
		Where("members.user_id = ? AND members.status = ?", userID, "active").
		Scan(&rows).Error
	if err != nil {
		return false, err
	}
	for _, row := range rows {
		if memberScopeMatches(row, spaceID, targetOrgID) {
			return true, nil
		}
	}
	return false, nil
}

func hashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(b), err
}

func checkPasswordHash(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func setIdentity(c *gin.Context, actorID, spaceID, role string) {
	c.Set(ctxActorID, actorID)
	c.Set(ctxSpaceID, spaceID)
	c.Set(ctxRole, role)
}

func devSpaceOverride(c *gin.Context) string {
	if spaceID := strings.TrimSpace(c.GetHeader("X-ASH-Space-ID")); spaceID != "" {
		return spaceID
	}
	return "local"
}

func currentActor(c *gin.Context) string {
	if v, ok := c.Get(ctxActorID); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func currentSpace(c *gin.Context) string {
	if v, ok := c.Get(ctxSpaceID); ok {
		if s, ok := v.(string); ok && s != "" {
			return s
		}
	}
	return "local"
}

func requireRole(c *gin.Context, allowed ...string) bool {
	role := ""
	if v, ok := c.Get(ctxRole); ok {
		role, _ = v.(string)
	}
	for _, item := range allowed {
		if role == item {
			return true
		}
	}
	c.AbortWithStatusJSON(http.StatusForbidden, errorBody("FORBIDDEN", "insufficient role"))
	return false
}

func (h *Handler) requirePermission(c *gin.Context, permission string, spaceIDs ...string) bool {
	spaceID := currentSpace(c)
	if len(spaceIDs) > 0 && strings.TrimSpace(spaceIDs[0]) != "" {
		spaceID = strings.TrimSpace(spaceIDs[0])
	}
	ok, err := h.hasPermission(c, spaceID, permission)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, errorBody("PERMISSION_CHECK_FAILED", err.Error()))
		return false
	}
	if ok {
		return true
	}
	c.AbortWithStatusJSON(http.StatusForbidden, errorBody("FORBIDDEN", "missing permission "+permission))
	return false
}

func (h *Handler) requireRunPermission(c *gin.Context, runID, permission string) bool {
	if !h.requireRunAccess(c, runID) {
		return false
	}
	spaceID, err := h.runSpaceID(c, runID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, errorBody("RUN_SCOPE_CHECK_FAILED", err.Error()))
		return false
	}
	return h.requirePermission(c, permission, spaceID)
}

func (h *Handler) requireOrgPermission(c *gin.Context, orgID, permission string) bool {
	ok, err := h.hasOrgPermission(c, orgID, permission)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, errorBody("PERMISSION_CHECK_FAILED", err.Error()))
		return false
	}
	if ok {
		return true
	}
	c.AbortWithStatusJSON(http.StatusForbidden, errorBody("FORBIDDEN", "missing permission "+permission))
	return false
}

func (h *Handler) hasPermission(c *gin.Context, spaceID, permission string) (bool, error) {
	if roleAllowsPermission(currentRole(c), permission) {
		return true, nil
	}
	actor := currentActor(c)
	if actor == "" {
		return false, nil
	}
	perms, err := h.memberPermissions(c, actor, spaceID)
	if err != nil {
		return false, err
	}
	for _, item := range perms {
		if permissionMatches(item, permission) {
			return true, nil
		}
	}
	return false, nil
}

func (h *Handler) hasOrgPermission(c *gin.Context, orgID, permission string) (bool, error) {
	if roleAllowsPermission(currentRole(c), permission) {
		return true, nil
	}
	actor := currentActor(c)
	if actor == "" {
		return false, nil
	}
	perms, err := h.orgMemberPermissions(c, actor, orgID)
	if err != nil {
		return false, err
	}
	for _, item := range perms {
		if permissionMatches(item, permission) {
			return true, nil
		}
	}
	return false, nil
}

func currentRole(c *gin.Context) string {
	if v, ok := c.Get(ctxRole); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func roleAllowsPermission(role, permission string) bool {
	if authz.RoleAllows(role, permission) {
		return true
	}
	// Legacy JWT role aliases
	switch role {
	case "admin":
		return true
	default:
		return false
	}
}

type memberPermissionRow struct {
	OrgID       string
	SpaceID     string
	Permissions string
}

func (h *Handler) memberPermissions(c *gin.Context, actorID, targetSpaceID string) ([]string, error) {
	var rows []memberPermissionRow
	err := h.dbBypass(c).Table("members").
		Select("members.org_id, members.space_id, roles.permissions").
		Joins("JOIN roles ON roles.id = members.role_id").
		Where("members.user_id = ? AND members.status = ?", actorID, "active").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	targetOrgID, err := h.spaceOrgID(c, targetSpaceID)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0)
	for _, row := range rows {
		if !memberScopeMatches(row, targetSpaceID, targetOrgID) {
			continue
		}
		out = append(out, parsePermissions(row.Permissions)...)
	}
	return out, nil
}

func (h *Handler) orgMemberPermissions(c *gin.Context, actorID, orgID string) ([]string, error) {
	var rows []memberPermissionRow
	err := h.dbBypass(c).Table("members").
		Select("members.org_id, members.space_id, roles.permissions").
		Joins("JOIN roles ON roles.id = members.role_id").
		Where("members.user_id = ? AND members.status = ? AND members.org_id = ? AND (members.space_id = '' OR members.space_id IS NULL)", actorID, "active", orgID).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]string, 0)
	for _, row := range rows {
		out = append(out, parsePermissions(row.Permissions)...)
	}
	return out, nil
}

func (h *Handler) spaceOrgID(c *gin.Context, spaceID string) (string, error) {
	if spaceID == "" || spaceID == "local" {
		return "", nil
	}
	var row struct {
		OrgID string
	}
	if err := h.dbBypass(c).Table("spaces").Select("org_id").Where("id = ?", spaceID).Scan(&row).Error; err != nil {
		return "", err
	}
	return row.OrgID, nil
}

func (h *Handler) runSpaceID(c *gin.Context, runID string) (string, error) {
	var row struct {
		SpaceID string
	}
	if err := h.dbBypass(c).Table("runs").Select("space_id").Where("id = ?", runID).Take(&row).Error; err != nil {
		return "", err
	}
	return firstNonEmptyAPI(row.SpaceID, "local"), nil
}

func memberScopeMatches(row memberPermissionRow, targetSpaceID, targetOrgID string) bool {
	if targetSpaceID == "" {
		targetSpaceID = "local"
	}
	if row.SpaceID != "" {
		return row.SpaceID == targetSpaceID
	}
	if targetSpaceID == "local" {
		return row.OrgID == "" || row.OrgID == "local"
	}
	return row.OrgID != "" && row.OrgID == targetOrgID
}

func parsePermissions(raw string) []string {
	var values []string
	if err := json.Unmarshal([]byte(raw), &values); err == nil {
		return values
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	return []string{raw}
}

func permissionMatches(grant, want string) bool {
	grant = strings.TrimSpace(grant)
	want = strings.TrimSpace(want)
	if grant == "" || want == "" {
		return false
	}
	if grant == "*" || grant == want {
		return true
	}
	if strings.HasSuffix(grant, ":*") {
		return strings.HasPrefix(want, strings.TrimSuffix(grant, "*"))
	}
	return false
}

func (h *Handler) requireRunAccess(c *gin.Context, runID string) bool {
	sum, err := h.runsFor(c).Get(runID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, errorBody("RUN_NOT_FOUND", "run not found"))
		return false
	}
	if err := store.EnforceSpaceAccess(sum.SpaceID, currentSpace(c)); err != nil {
		c.AbortWithStatusJSON(http.StatusForbidden, errorBody("SPACE_ACCESS_DENIED", err.Error()))
		return false
	}
	return true
}

func signToken(claims tokenClaims, secret string) (string, error) {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	bodyBytes, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	body := base64.RawURLEncoding.EncodeToString(bodyBytes)
	sig := sign(header+"."+body, secret)
	return header + "." + body + "." + sig, nil
}

func verifyToken(token, secret string) (*tokenClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid token shape")
	}
	want := sign(parts[0]+"."+parts[1], secret)
	if !hmac.Equal([]byte(want), []byte(parts[2])) {
		return nil, fmt.Errorf("invalid token signature")
	}
	body, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid token payload")
	}
	var claims tokenClaims
	if err := json.Unmarshal(body, &claims); err != nil {
		return nil, fmt.Errorf("invalid token claims")
	}
	if claims.Exp > 0 && time.Now().Unix() > claims.Exp {
		return nil, fmt.Errorf("token expired")
	}
	if claims.Sub == "" {
		return nil, fmt.Errorf("token subject is required")
	}
	return &claims, nil
}

func sign(value, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(value))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func bearerToken(header string) string {
	if header == "" {
		return ""
	}
	parts := strings.Fields(header)
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return parts[1]
	}
	return ""
}
