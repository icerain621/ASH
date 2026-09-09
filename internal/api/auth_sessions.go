package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ash-repwiki/ash/internal/config"
	"github.com/ash-repwiki/ash/internal/store"
)

const (
	authSessionEventType   = "auth.session"
	authSessionTypPrimary  = "primary"
	authSessionTypDevice   = "device"
	authSessionStatusActive  = "active"
	authSessionStatusRevoked = "revoked"
	authSessionPrimaryTTL  = 24 * time.Hour
	authSessionDeviceTTL   = 4 * time.Hour
)

type authSessionPayload struct {
	SID         string   `json:"sid"`
	UserID      string   `json:"userId"`
	SpaceID     string   `json:"spaceId"`
	DID         string   `json:"did"`
	Typ         string   `json:"typ"`
	Scope       []string `json:"scope,omitempty"`
	Status      string   `json:"status"`
	Exp         int64    `json:"exp"`
	ParentSID   string   `json:"parentSid,omitempty"`
	UserAgent   string   `json:"userAgent,omitempty"`
	RotatedAt   int64    `json:"rotatedAt,omitempty"`
	RotateCount int      `json:"rotateCount,omitempty"`
}

type createDeviceSessionRequest struct {
	DeviceID    string   `json:"deviceId"`
	DeviceLabel string   `json:"deviceLabel"`
	Scope       []string `json:"scope"`
	TTLSeconds  int      `json:"ttlSeconds"`
}

type refreshAuthSessionRequest struct {
	TTLSeconds int `json:"ttlSeconds"`
}

type authSessionListResponse struct {
	Items []AuthGatewaySession `json:"items"`
}

// CreateDeviceAuthSession godoc
// @Summary Mint a device-scoped auth token from a primary session
// @Tags auth
// @Accept json
// @Produce json
// @Param body body createDeviceSessionRequest false "device session"
// @Success 200 {object} AuthSessionResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 401 {object} APIErrorResponse
// @Failure 403 {object} APIErrorResponse
// @Router /api/v1/auth/sessions/device [post]
func (h *Handler) createDeviceAuthSession(c *gin.Context) {
	actorID := currentActor(c)
	if actorID == "" {
		c.JSON(http.StatusUnauthorized, errorBody("UNAUTHORIZED", "missing authenticated actor"))
		return
	}
	typ, _ := c.Get(ctxTokenTyp)
	typStr, _ := typ.(string)
	if typStr != "" && typStr != authSessionTypPrimary {
		c.JSON(http.StatusForbidden, errorBody("AUTH_SESSION_PRIMARY_REQUIRED", "device tokens must be minted from a primary session"))
		return
	}
	var req createDeviceSessionRequest
	_ = c.ShouldBindJSON(&req)
	did := strings.TrimSpace(req.DeviceID)
	if did == "" {
		did = strings.TrimSpace(req.DeviceLabel)
	}
	if did == "" {
		did = "did_" + uuid.NewString()
	}
	ttl := authSessionDeviceTTL
	if req.TTLSeconds > 0 {
		ttl = clampAuthSessionTTL(req.TTLSeconds)
	}
	if len(req.Scope) > 0 {
		if ok, err := h.callerGrantsCoverScope(c, currentSpace(c), req.Scope); err != nil {
			c.JSON(http.StatusInternalServerError, errorBody("PERMISSION_CHECK_FAILED", err.Error()))
			return
		} else if !ok {
			c.JSON(http.StatusBadRequest, errorBody("AUTH_SCOPE_INVALID", "device scope exceeds caller permissions"))
			return
		}
	}
	parentSID, _ := c.Get(ctxSessionID)
	parentStr, _ := parentSID.(string)
	email, name := "", ""
	var u store.User
	if err := h.dbBypass(c).First(&u, "id = ?", actorID).Error; err == nil {
		email, name = u.Email, u.DisplayName
	}
	token, sess, user, space, err := h.issueAuthSession(c, issueAuthSessionOpts{
		UserID:    actorID,
		SpaceID:   currentSpace(c),
		Role:      firstNonEmptyAPI(currentRole(c), "viewer"),
		Typ:       authSessionTypDevice,
		DID:       did,
		Scope:     req.Scope,
		TTL:       ttl,
		ParentSID: parentStr,
		Email:     email,
		Name:      name,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("AUTH_SESSION_CREATE_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, AuthSessionResponse{Token: token, User: user, Space: space, Session: sess})
}

// ListAuthSessions godoc
// @Summary List auth sessions for the current user
// @Tags auth
// @Produce json
// @Success 200 {object} authSessionListResponse
// @Failure 401 {object} APIErrorResponse
// @Router /api/v1/auth/sessions [get]
func (h *Handler) listAuthSessions(c *gin.Context) {
	actorID := currentActor(c)
	if actorID == "" {
		c.JSON(http.StatusUnauthorized, errorBody("UNAUTHORIZED", "missing authenticated actor"))
		return
	}
	var rows []store.AuditLog
	err := h.dbBypass(c).Where("event_type = ? AND actor_id = ?", authSessionEventType, actorID).
		Order("created_at desc").Limit(100).Find(&rows).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("AUTH_SESSION_LOOKUP_FAILED", err.Error()))
		return
	}
	items := make([]AuthGatewaySession, 0, len(rows))
	for _, row := range rows {
		if view, err := decodeAuthSessionPayload(row); err == nil {
			items = append(items, *view)
		}
	}
	c.JSON(http.StatusOK, authSessionListResponse{Items: items})
}

// RevokeAuthSession godoc
// @Summary Revoke an auth session by sid
// @Tags auth
// @Produce json
// @Param sid path string true "auth session id"
// @Success 200 {object} map[string]any
// @Failure 401 {object} APIErrorResponse
// @Failure 403 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Router /api/v1/auth/sessions/{sid} [delete]
func (h *Handler) revokeAuthSession(c *gin.Context) {
	actorID := currentActor(c)
	if actorID == "" {
		c.JSON(http.StatusUnauthorized, errorBody("UNAUTHORIZED", "missing authenticated actor"))
		return
	}
	sid := strings.TrimSpace(c.Param("sid"))
	row, payload, err := h.loadAuthSession(c, sid)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, errorBody("AUTH_SESSION_NOT_FOUND", "auth session not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, errorBody("AUTH_SESSION_LOOKUP_FAILED", err.Error()))
		return
	}
	if payload.UserID != actorID && currentRole(c) != "admin" {
		c.JSON(http.StatusForbidden, errorBody("AUTH_SESSION_FORBIDDEN", "cannot revoke another user's session"))
		return
	}
	payload.Status = authSessionStatusRevoked
	b, _ := json.Marshal(payload)
	if err := h.dbBypass(c).Model(&store.AuditLog{}).Where("id = ?", row.ID).Update("payload_json", string(b)).Error; err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("AUTH_SESSION_REVOKE_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "sid": sid, "status": authSessionStatusRevoked})
}

// RefreshAuthSession godoc
// @Summary Rotate access JWT for an active auth session (same sid)
// @Tags auth
// @Accept json
// @Produce json
// @Param body body refreshAuthSessionRequest false "optional ttl"
// @Success 200 {object} AuthSessionResponse
// @Failure 401 {object} APIErrorResponse
// @Failure 403 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Router /api/v1/auth/sessions/refresh [post]
func (h *Handler) refreshAuthSession(c *gin.Context) {
	if expired, _ := c.Get(ctxTokenExpired); expired == true {
		c.JSON(http.StatusUnauthorized, errorBody("AUTH_SESSION_EXPIRED", "auth session token has expired"))
		return
	}
	actorID := currentActor(c)
	if actorID == "" {
		c.JSON(http.StatusUnauthorized, errorBody("UNAUTHORIZED", "missing authenticated actor"))
		return
	}
	sidVal, _ := c.Get(ctxSessionID)
	sid, _ := sidVal.(string)
	if strings.TrimSpace(sid) == "" {
		c.JSON(http.StatusBadRequest, errorBody("AUTH_SESSION_NOT_FOUND", "token has no sid"))
		return
	}
	var req refreshAuthSessionRequest
	_ = c.ShouldBindJSON(&req)
	row, payload, err := h.loadAuthSession(c, sid)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, errorBody("AUTH_SESSION_NOT_FOUND", "auth session not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, errorBody("AUTH_SESSION_LOOKUP_FAILED", err.Error()))
		return
	}
	if payload.Status == authSessionStatusRevoked {
		c.JSON(http.StatusUnauthorized, errorBody("AUTH_SESSION_REVOKED", "auth session has been revoked"))
		return
	}
	if payload.UserID != actorID && currentRole(c) != "admin" {
		c.JSON(http.StatusForbidden, errorBody("AUTH_SESSION_FORBIDDEN", "cannot refresh another user's session"))
		return
	}
	ttl := time.Duration(0)
	if req.TTLSeconds > 0 {
		ttl = clampAuthSessionTTL(req.TTLSeconds)
	} else if payload.Typ == authSessionTypDevice {
		ttl = authSessionDeviceTTL
	} else {
		ttl = authSessionPrimaryTTL
	}
	email, name := "", ""
	var u store.User
	if err := h.dbBypass(c).First(&u, "id = ?", actorID).Error; err == nil {
		email, name = u.Email, u.DisplayName
	}
	token, sess, user, space, err := h.rotateAuthSession(c, row, payload, issueAuthSessionOpts{
		UserID: actorID,
		SpaceID: firstNonEmptyAPI(payload.SpaceID, currentSpace(c)),
		Role:   firstNonEmptyAPI(currentRole(c), "viewer"),
		Typ:    firstNonEmptyAPI(payload.Typ, authSessionTypPrimary),
		DID:    payload.DID,
		Scope:  append([]string(nil), payload.Scope...),
		TTL:    ttl,
		Email:  email,
		Name:   name,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("AUTH_SESSION_CREATE_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, AuthSessionResponse{Token: token, User: user, Space: space, Session: sess})
}

func clampAuthSessionTTL(sec int) time.Duration {
	if sec < 300 {
		sec = 300
	}
	if sec > 86400 {
		sec = 86400
	}
	return time.Duration(sec) * time.Second
}

func (h *Handler) callerGrantsCoverScope(c *gin.Context, spaceID string, scope []string) (bool, error) {
	for _, item := range scope {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		ok, err := h.hasPermission(c, spaceID, item)
		if err != nil {
			return false, err
		}
		if !ok {
			return false, nil
		}
	}
	return true, nil
}

type issueAuthSessionOpts struct {
	UserID    string
	SpaceID   string
	Role      string
	Typ       string
	DID       string
	Scope     []string
	TTL       time.Duration
	ParentSID string
	Email     string
	Name      string
	UserAgent string
}

func (h *Handler) issueAuthSession(c *gin.Context, opt issueAuthSessionOpts) (string, *AuthGatewaySession, AuthUser, AuthSpace, error) {
	if opt.Typ == "" {
		opt.Typ = authSessionTypPrimary
	}
	if opt.TTL <= 0 {
		if opt.Typ == authSessionTypDevice {
			opt.TTL = authSessionDeviceTTL
		} else {
			opt.TTL = authSessionPrimaryTTL
		}
	}
	if opt.DID == "" {
		opt.DID = "did_primary"
	}
	sid := "asess_" + uuid.NewString()
	exp := time.Now().UTC().Add(opt.TTL).Unix()
	claims := tokenClaims{
		Sub: opt.UserID, SpaceID: opt.SpaceID, Role: opt.Role, Exp: exp,
		Sid: sid, Did: opt.DID, Typ: opt.Typ, Scope: opt.Scope,
	}
	cfg := config.Load()
	token, err := signToken(claims, cfg.JWTSecret)
	if err != nil {
		return "", nil, AuthUser{}, AuthSpace{}, err
	}
	ua := opt.UserAgent
	if ua == "" && c != nil {
		ua = c.GetHeader("User-Agent")
	}
	payload := authSessionPayload{
		SID: sid, UserID: opt.UserID, SpaceID: opt.SpaceID, DID: opt.DID, Typ: opt.Typ,
		Scope: opt.Scope, Status: authSessionStatusActive, Exp: exp, ParentSID: opt.ParentSID, UserAgent: ua,
	}
	b, _ := json.Marshal(payload)
	row := &store.AuditLog{
		ID: sid, SpaceID: firstNonEmptyAPI(opt.SpaceID, "local"), ActorID: opt.UserID,
		EventType: authSessionEventType, PayloadJSON: string(b), CreatedAt: time.Now().UTC(),
	}
	if err := h.dbBypass(c).Create(row).Error; err != nil {
		return "", nil, AuthUser{}, AuthSpace{}, err
	}
	return token, sessionViewFrom(opt, sid, exp), authUserFrom(opt), authSpaceFrom(c, h, opt.SpaceID), nil
}

func (h *Handler) rotateAuthSession(c *gin.Context, row store.AuditLog, payload authSessionPayload, opt issueAuthSessionOpts) (string, *AuthGatewaySession, AuthUser, AuthSpace, error) {
	if opt.TTL <= 0 {
		if opt.Typ == authSessionTypDevice {
			opt.TTL = authSessionDeviceTTL
		} else {
			opt.TTL = authSessionPrimaryTTL
		}
	}
	now := time.Now().UTC()
	exp := now.Add(opt.TTL).Unix()
	claims := tokenClaims{
		Sub: opt.UserID, SpaceID: opt.SpaceID, Role: opt.Role, Exp: exp,
		Sid: payload.SID, Did: opt.DID, Typ: opt.Typ, Scope: opt.Scope,
	}
	cfg := config.Load()
	token, err := signToken(claims, cfg.JWTSecret)
	if err != nil {
		return "", nil, AuthUser{}, AuthSpace{}, err
	}
	payload.Exp = exp
	payload.RotatedAt = now.Unix()
	payload.RotateCount++
	payload.Scope = opt.Scope
	payload.DID = opt.DID
	payload.Typ = opt.Typ
	payload.SpaceID = opt.SpaceID
	b, _ := json.Marshal(payload)
	if err := h.dbBypass(c).Model(&store.AuditLog{}).Where("id = ?", row.ID).Update("payload_json", string(b)).Error; err != nil {
		return "", nil, AuthUser{}, AuthSpace{}, err
	}
	sess := &AuthGatewaySession{
		SID: payload.SID, DID: opt.DID, Typ: opt.Typ, Scope: opt.Scope,
		Status: authSessionStatusActive, Exp: exp, SpaceID: opt.SpaceID,
	}
	return token, sess, authUserFrom(opt), authSpaceFrom(c, h, opt.SpaceID), nil
}

func sessionViewFrom(opt issueAuthSessionOpts, sid string, exp int64) *AuthGatewaySession {
	return &AuthGatewaySession{
		SID: sid, DID: opt.DID, Typ: opt.Typ, Scope: opt.Scope,
		Status: authSessionStatusActive, Exp: exp, SpaceID: opt.SpaceID,
	}
}

func authUserFrom(opt issueAuthSessionOpts) AuthUser {
	return AuthUser{ID: opt.UserID, Email: opt.Email, DisplayName: opt.Name}
}

func authSpaceFrom(c *gin.Context, h *Handler, spaceID string) AuthSpace {
	spaceName := "Local"
	if spaceID != "" && spaceID != "local" {
		var space store.Space
		if err := h.dbBypass(c).First(&space, "id = ?", spaceID).Error; err == nil {
			spaceName = space.Name
		}
	}
	return AuthSpace{ID: firstNonEmptyAPI(spaceID, "local"), Name: spaceName}
}

func (h *Handler) loadAuthSession(c *gin.Context, sid string) (store.AuditLog, authSessionPayload, error) {
	sid = strings.TrimSpace(sid)
	var row store.AuditLog
	if err := h.dbBypass(c).First(&row, "id = ? AND event_type = ?", sid, authSessionEventType).Error; err != nil {
		return store.AuditLog{}, authSessionPayload{}, err
	}
	var payload authSessionPayload
	if err := json.Unmarshal([]byte(row.PayloadJSON), &payload); err != nil {
		return store.AuditLog{}, authSessionPayload{}, err
	}
	return row, payload, nil
}

func (h *Handler) authSessionRevoked(c *gin.Context, sid string) (bool, error) {
	_, payload, err := h.loadAuthSession(c, sid)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Session claim without registry row: treat as non-revoked for backward compat.
			return false, nil
		}
		return false, err
	}
	return payload.Status == authSessionStatusRevoked, nil
}

func decodeAuthSessionPayload(row store.AuditLog) (*AuthGatewaySession, error) {
	var payload authSessionPayload
	if err := json.Unmarshal([]byte(row.PayloadJSON), &payload); err != nil {
		return nil, err
	}
	return &AuthGatewaySession{
		SID: payload.SID, DID: payload.DID, Typ: payload.Typ, Scope: payload.Scope,
		Status: payload.Status, Exp: payload.Exp, SpaceID: payload.SpaceID,
	}, nil
}
