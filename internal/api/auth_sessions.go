package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ash-repwiki/ash/internal/config"
	"github.com/ash-repwiki/ash/internal/store"
)

const (
	authSessionEventType     = "auth.session"
	authSessionTypPrimary    = "primary"
	authSessionTypDevice     = "device"
	authSessionTypRefresh    = "refresh"
	authSessionStatusActive  = "active"
	authSessionStatusRevoked = "revoked"
	authSessionPrimaryTTL    = 24 * time.Hour
	authSessionDeviceTTL     = 4 * time.Hour
	authSessionRefreshTTLMin = 24 * time.Hour
	authSessionRefreshTTLMax = 90 * 24 * time.Hour
	authSessionRefreshTTLDef = 30 * 24 * time.Hour
)

type authSessionPayload struct {
	SID            string   `json:"sid"`
	UserID         string   `json:"userId"`
	SpaceID        string   `json:"spaceId"`
	DID            string   `json:"did"`
	Typ            string   `json:"typ"`
	Scope          []string `json:"scope,omitempty"`
	Status         string   `json:"status"`
	Exp            int64    `json:"exp"`
	RefreshExp     int64    `json:"refreshExp,omitempty"`
	AccessJti      string   `json:"accessJti,omitempty"`
	RefreshJti     string   `json:"refreshJti,omitempty"`
	PrevAccessJti  string   `json:"prevAccessJti,omitempty"`
	PrevRefreshJti string   `json:"prevRefreshJti,omitempty"`
	ParentSID      string   `json:"parentSid,omitempty"`
	UserAgent      string   `json:"userAgent,omitempty"`
	RotatedAt      int64    `json:"rotatedAt,omitempty"`
	RotateCount    int      `json:"rotateCount,omitempty"`
}

type createDeviceSessionRequest struct {
	DeviceID    string   `json:"deviceId"`
	DeviceLabel string   `json:"deviceLabel"`
	Scope       []string `json:"scope"`
	TTLSeconds  int      `json:"ttlSeconds"`
}

type refreshAuthSessionRequest struct {
	RefreshToken string `json:"refreshToken"`
	TTLSeconds   int    `json:"ttlSeconds"`
}

type authSessionListResponse struct {
	Items []AuthGatewaySession `json:"items"`
}

type issuedAuthTokens struct {
	AccessToken  string
	RefreshToken string
	Session      *AuthGatewaySession
	User         AuthUser
	Space        AuthSpace
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
	issued, err := h.issueAuthSession(c, issueAuthSessionOpts{
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
	c.JSON(http.StatusOK, authSessionResponseFrom(issued))
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
// @Summary Rotate access+refresh JWTs for an active auth session (same sid)
// @Tags auth
// @Accept json
// @Produce json
// @Param body body refreshAuthSessionRequest false "refreshToken and optional access ttl"
// @Success 200 {object} AuthSessionResponse
// @Failure 401 {object} APIErrorResponse
// @Failure 403 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Router /api/v1/auth/sessions/refresh [post]
func (h *Handler) refreshAuthSession(c *gin.Context) {
	var req refreshAuthSessionRequest
	_ = c.ShouldBindJSON(&req)
	raw := strings.TrimSpace(req.RefreshToken)
	if raw == "" {
		raw = bearerToken(c.GetHeader("Authorization"))
	}
	if raw == "" {
		c.JSON(http.StatusUnauthorized, errorBody("UNAUTHORIZED", "missing refresh token"))
		return
	}
	cfg := config.Load()
	claims, err := verifyToken(raw, cfg.JWTSecret)
	if err != nil {
		if _, err2 := verifyTokenIgnoreExp(raw, cfg.JWTSecret); err2 == nil {
			c.JSON(http.StatusUnauthorized, errorBody("AUTH_REFRESH_REQUIRED", "use a refresh token to renew the session"))
			return
		}
		c.JSON(http.StatusUnauthorized, errorBody("UNAUTHORIZED", err.Error()))
		return
	}
	typ := firstNonEmptyAPI(claims.Typ, authSessionTypPrimary)
	switch typ {
	case authSessionTypRefresh:
		// preferred path
	case authSessionTypPrimary, authSessionTypDevice:
		// legacy: unexpired access may still refresh
	default:
		c.JSON(http.StatusUnauthorized, errorBody("AUTH_REFRESH_REQUIRED", "use a refresh token to renew the session"))
		return
	}
	sid := strings.TrimSpace(claims.Sid)
	if sid == "" {
		c.JSON(http.StatusBadRequest, errorBody("AUTH_SESSION_NOT_FOUND", "token has no sid"))
		return
	}
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
	if !authSessionAllowsJti(payload, claims, time.Now().UTC(), authTokenGrace()) {
		c.JSON(http.StatusUnauthorized, errorBody("AUTH_TOKEN_REPLAY", "token jti is no longer current"))
		return
	}
	if payload.UserID != claims.Sub && claims.Role != "admin" {
		c.JSON(http.StatusForbidden, errorBody("AUTH_SESSION_FORBIDDEN", "cannot refresh another user's session"))
		return
	}
	accessTyp := firstNonEmptyAPI(payload.Typ, authSessionTypPrimary)
	if accessTyp == authSessionTypRefresh {
		accessTyp = authSessionTypPrimary
	}
	ttl := time.Duration(0)
	if req.TTLSeconds > 0 {
		ttl = clampAuthSessionTTL(req.TTLSeconds)
	} else if accessTyp == authSessionTypDevice {
		ttl = authSessionDeviceTTL
	} else {
		ttl = authSessionPrimaryTTL
	}
	email, name := "", ""
	var u store.User
	if err := h.dbBypass(c).First(&u, "id = ?", claims.Sub).Error; err == nil {
		email, name = u.Email, u.DisplayName
	}
	issued, err := h.rotateAuthSession(c, row, payload, issueAuthSessionOpts{
		UserID:  claims.Sub,
		SpaceID: firstNonEmptyAPI(payload.SpaceID, claims.SpaceID, "local"),
		Role:    firstNonEmptyAPI(claims.Role, "viewer"),
		Typ:     accessTyp,
		DID:     firstNonEmptyAPI(payload.DID, claims.Did),
		Scope:   append([]string(nil), payload.Scope...),
		TTL:     ttl,
		Email:   email,
		Name:    name,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("AUTH_SESSION_CREATE_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, authSessionResponseFrom(issued))
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

func authRefreshTTL() time.Duration {
	raw := strings.TrimSpace(os.Getenv("ASH_AUTH_REFRESH_TTL_SEC"))
	if raw == "" {
		return authSessionRefreshTTLDef
	}
	sec, err := strconv.Atoi(raw)
	if err != nil || sec <= 0 {
		return authSessionRefreshTTLDef
	}
	d := time.Duration(sec) * time.Second
	if d < authSessionRefreshTTLMin {
		return authSessionRefreshTTLMin
	}
	if d > authSessionRefreshTTLMax {
		return authSessionRefreshTTLMax
	}
	return d
}

const (
	authTokenGraceDefault = 60 * time.Second
	authTokenGraceMax     = 300 * time.Second
)

func authTokenGrace() time.Duration {
	raw := strings.TrimSpace(os.Getenv("ASH_AUTH_TOKEN_GRACE_SEC"))
	if raw == "" {
		return authTokenGraceDefault
	}
	sec, err := strconv.Atoi(raw)
	if err != nil {
		return authTokenGraceDefault
	}
	if sec <= 0 {
		return 0
	}
	d := time.Duration(sec) * time.Second
	if d > authTokenGraceMax {
		return authTokenGraceMax
	}
	return d
}

func authSessionAllowsJti(payload authSessionPayload, claims *tokenClaims, now time.Time, grace time.Duration) bool {
	if claims == nil || strings.TrimSpace(claims.Jti) == "" {
		return true
	}
	// Pre-DX63 registry rows: unbound until issue/rotate stores jtis.
	if payload.AccessJti == "" && payload.RefreshJti == "" {
		return true
	}
	cur, prev := payload.AccessJti, payload.PrevAccessJti
	if claims.Typ == authSessionTypRefresh {
		cur, prev = payload.RefreshJti, payload.PrevRefreshJti
	}
	jti := strings.TrimSpace(claims.Jti)
	if jti == cur {
		return true
	}
	if prev != "" && jti == prev && payload.RotatedAt > 0 {
		if grace <= 0 {
			return false
		}
		deadline := payload.RotatedAt + int64(grace/time.Second)
		return now.Unix() <= deadline
	}
	return false
}

func authSessionResponseFrom(issued issuedAuthTokens) AuthSessionResponse {
	return AuthSessionResponse{
		Token:        issued.AccessToken,
		RefreshToken: issued.RefreshToken,
		User:         issued.User,
		Space:        issued.Space,
		Session:      issued.Session,
	}
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

func (h *Handler) issueAuthSession(c *gin.Context, opt issueAuthSessionOpts) (issuedAuthTokens, error) {
	var empty issuedAuthTokens
	if opt.Typ == "" {
		opt.Typ = authSessionTypPrimary
	}
	if opt.Typ == authSessionTypRefresh {
		return empty, errors.New("cannot issue standalone refresh session registry")
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
	now := time.Now().UTC()
	exp := now.Add(opt.TTL).Unix()
	refreshExp := now.Add(authRefreshTTL()).Unix()
	cfg := config.Load()
	accessJti := uuid.NewString()
	refreshJti := uuid.NewString()
	access, err := signToken(tokenClaims{
		Sub: opt.UserID, SpaceID: opt.SpaceID, Role: opt.Role, Exp: exp, Iat: now.Unix(),
		Jti: accessJti, Sid: sid, Did: opt.DID, Typ: opt.Typ, Scope: opt.Scope,
	}, cfg.JWTSecret)
	if err != nil {
		return empty, err
	}
	refresh, err := signToken(tokenClaims{
		Sub: opt.UserID, SpaceID: opt.SpaceID, Role: opt.Role, Exp: refreshExp, Iat: now.Unix(),
		Jti: refreshJti, Sid: sid, Did: opt.DID, Typ: authSessionTypRefresh, Scope: opt.Scope,
	}, cfg.JWTSecret)
	if err != nil {
		return empty, err
	}
	ua := opt.UserAgent
	if ua == "" && c != nil {
		ua = c.GetHeader("User-Agent")
	}
	payload := authSessionPayload{
		SID: sid, UserID: opt.UserID, SpaceID: opt.SpaceID, DID: opt.DID, Typ: opt.Typ,
		Scope: opt.Scope, Status: authSessionStatusActive, Exp: exp, RefreshExp: refreshExp,
		AccessJti: accessJti, RefreshJti: refreshJti,
		ParentSID: opt.ParentSID, UserAgent: ua,
	}
	b, _ := json.Marshal(payload)
	row := &store.AuditLog{
		ID: sid, SpaceID: firstNonEmptyAPI(opt.SpaceID, "local"), ActorID: opt.UserID,
		EventType: authSessionEventType, PayloadJSON: string(b), CreatedAt: now,
	}
	if err := h.dbBypass(c).Create(row).Error; err != nil {
		return empty, err
	}
	return issuedAuthTokens{
		AccessToken:  access,
		RefreshToken: refresh,
		Session:      sessionViewFrom(opt, sid, exp),
		User:         authUserFrom(opt),
		Space:        authSpaceFrom(c, h, opt.SpaceID),
	}, nil
}

func (h *Handler) rotateAuthSession(c *gin.Context, row store.AuditLog, payload authSessionPayload, opt issueAuthSessionOpts) (issuedAuthTokens, error) {
	var empty issuedAuthTokens
	if opt.Typ == "" || opt.Typ == authSessionTypRefresh {
		opt.Typ = firstNonEmptyAPI(payload.Typ, authSessionTypPrimary)
		if opt.Typ == authSessionTypRefresh {
			opt.Typ = authSessionTypPrimary
		}
	}
	if opt.TTL <= 0 {
		if opt.Typ == authSessionTypDevice {
			opt.TTL = authSessionDeviceTTL
		} else {
			opt.TTL = authSessionPrimaryTTL
		}
	}
	now := time.Now().UTC()
	exp := now.Add(opt.TTL).Unix()
	refreshExp := now.Add(authRefreshTTL()).Unix()
	cfg := config.Load()
	accessJti := uuid.NewString()
	refreshJti := uuid.NewString()
	access, err := signToken(tokenClaims{
		Sub: opt.UserID, SpaceID: opt.SpaceID, Role: opt.Role, Exp: exp, Iat: now.Unix(),
		Jti: accessJti, Sid: payload.SID, Did: opt.DID, Typ: opt.Typ, Scope: opt.Scope,
	}, cfg.JWTSecret)
	if err != nil {
		return empty, err
	}
	refresh, err := signToken(tokenClaims{
		Sub: opt.UserID, SpaceID: opt.SpaceID, Role: opt.Role, Exp: refreshExp, Iat: now.Unix(),
		Jti: refreshJti, Sid: payload.SID, Did: opt.DID, Typ: authSessionTypRefresh, Scope: opt.Scope,
	}, cfg.JWTSecret)
	if err != nil {
		return empty, err
	}
	payload.PrevAccessJti = payload.AccessJti
	payload.PrevRefreshJti = payload.RefreshJti
	payload.AccessJti = accessJti
	payload.RefreshJti = refreshJti
	payload.Exp = exp
	payload.RefreshExp = refreshExp
	payload.RotatedAt = now.Unix()
	payload.RotateCount++
	payload.Scope = opt.Scope
	payload.DID = opt.DID
	payload.Typ = opt.Typ
	payload.SpaceID = opt.SpaceID
	payload.Status = authSessionStatusActive
	b, _ := json.Marshal(payload)
	if err := h.dbBypass(c).Model(&store.AuditLog{}).Where("id = ?", row.ID).Update("payload_json", string(b)).Error; err != nil {
		return empty, err
	}
	return issuedAuthTokens{
		AccessToken:  access,
		RefreshToken: refresh,
		Session: &AuthGatewaySession{
			SID: payload.SID, DID: opt.DID, Typ: opt.Typ, Scope: opt.Scope,
			Status: authSessionStatusActive, Exp: exp, SpaceID: opt.SpaceID,
		},
		User:  authUserFrom(opt),
		Space: authSpaceFrom(c, h, opt.SpaceID),
	}, nil
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

// gateAuthSessionClaims enforces sid revoke + jti bind. Returns error code or "".
func (h *Handler) gateAuthSessionClaims(c *gin.Context, claims *tokenClaims) (code string, err error) {
	if claims == nil || strings.TrimSpace(claims.Sid) == "" {
		return "", nil
	}
	_, payload, err := h.loadAuthSession(c, claims.Sid)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil
		}
		return "AUTH_SESSION_LOOKUP_FAILED", err
	}
	if payload.Status == authSessionStatusRevoked {
		return "AUTH_SESSION_REVOKED", nil
	}
	if !authSessionAllowsJti(payload, claims, time.Now().UTC(), authTokenGrace()) {
		return "AUTH_TOKEN_REPLAY", nil
	}
	return "", nil
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
