package api

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ash-repwiki/ash/internal/idp"
	"github.com/ash-repwiki/ash/internal/store"
)

// OIDCLogin godoc
// @Summary Start OIDC authorization-code login
// @Description Redirects to the configured IdP when ASH_OIDC_ENABLED=1. Pass ui=1 for console hash redirect on callback.
// @Tags auth
// @Produce json
// @Param ui query string false "set to 1 for /ui/login hash redirect after callback"
// @Success 302 {string} string "redirect to IdP"
// @Failure 404 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/auth/oidc/login [get]
func (h *Handler) oidcLogin(c *gin.Context) {
	cli := h.oidcClient()
	if cli == nil || !cli.Enabled() {
		c.JSON(http.StatusNotFound, errorBody("OIDC_DISABLED", "OIDC is not enabled"))
		return
	}
	ui := strings.TrimSpace(c.Query("ui")) == "1" || strings.EqualFold(strings.TrimSpace(c.Query("ui")), "true")
	authURL, _, err := cli.BeginLogin(c.Request.Context(), ui)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("OIDC_LOGIN_FAILED", err.Error()))
		return
	}
	c.Redirect(http.StatusFound, authURL)
}

// OIDCCallback godoc
// @Summary OIDC authorization-code callback
// @Description Exchanges code for id_token claims, links/JIT user by oidc sub then email, returns ASH JWT session.
// @Tags auth
// @Produce json
// @Param code query string true "authorization code"
// @Param state query string true "csrf state"
// @Param spaceId query string false "preferred space id"
// @Success 200 {object} AuthSessionResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 403 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Failure 409 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/auth/oidc/callback [get]
func (h *Handler) oidcCallback(c *gin.Context) {
	cli := h.oidcClient()
	if cli == nil || !cli.Enabled() {
		c.JSON(http.StatusNotFound, errorBody("OIDC_DISABLED", "OIDC is not enabled"))
		return
	}
	state := strings.TrimSpace(c.Query("state"))
	code := strings.TrimSpace(c.Query("code"))
	if code == "" {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", "code query is required"))
		return
	}
	uiRedirect, ok := cli.ConsumeState(state)
	if !ok {
		c.JSON(http.StatusBadRequest, errorBody("OIDC_STATE_INVALID", "invalid or expired OIDC state"))
		return
	}
	claims, err := cli.ExchangeCode(c.Request.Context(), code)
	if err != nil {
		c.JSON(http.StatusUnauthorized, errorBody("OIDC_EXCHANGE_FAILED", err.Error()))
		return
	}
	user, err := h.ensureOIDCUser(c, claims)
	if err != nil {
		var linkErr *oidcLinkError
		if errors.As(err, &linkErr) {
			c.JSON(http.StatusConflict, errorBody("OIDC_LINK_CONFLICT", err.Error()))
			return
		}
		c.JSON(http.StatusInternalServerError, errorBody("OIDC_USER_FAILED", err.Error()))
		return
	}
	if user.Status != "" && user.Status != "active" {
		c.JSON(http.StatusForbidden, errorBody("USER_DISABLED", "user is not active"))
		return
	}
	spaceID := strings.TrimSpace(c.Query("spaceId"))
	if spaceID == "" {
		spaceID = strings.TrimSpace(os.Getenv("ASH_OIDC_DEFAULT_SPACE_ID"))
	}
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
	token, sess, userView, spaceView, err := h.issueAuthSession(c, issueAuthSessionOpts{
		UserID: user.ID, SpaceID: spaceID, Role: "viewer", Typ: authSessionTypPrimary,
		DID: "did_oidc", TTL: authSessionPrimaryTTL,
		Email: user.Email, Name: user.DisplayName,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("TOKEN_SIGN_FAILED", err.Error()))
		return
	}
	_ = h.dbBypass(c).Create(auditRow(spaceID, user.ID, "auth.oidc_login", map[string]any{
		"userId": user.ID, "email": user.Email, "spaceId": spaceID,
		"idpSub": claims.Subject, "idpIssuer": claims.Issuer, "sid": sess.SID,
	})).Error
	if uiRedirect {
		frag := url.Values{}
		frag.Set("token", token)
		frag.Set("spaceId", spaceView.ID)
		c.Redirect(http.StatusFound, "/ui/login#"+frag.Encode())
		return
	}
	c.JSON(http.StatusOK, AuthSessionResponse{
		Token: token, User: userView, Space: spaceView, Session: sess,
	})
}

func (h *Handler) oidcClient() *idp.Client {
	if h == nil {
		return nil
	}
	return h.oidc
}

type oidcLinkError struct{ msg string }

func (e *oidcLinkError) Error() string { return e.msg }

func (h *Handler) ensureOIDCUser(c *gin.Context, claims *idp.Claims) (store.User, error) {
	email := strings.TrimSpace(strings.ToLower(claims.Email))
	issuer := strings.TrimSpace(claims.Issuer)
	subject := strings.TrimSpace(claims.Subject)
	if email == "" {
		return store.User{}, fmt.Errorf("email required")
	}
	if issuer == "" || subject == "" {
		return store.User{}, fmt.Errorf("issuer and subject required")
	}

	db := h.dbBypass(c)
	var bySub store.User
	err := db.Where("oidc_issuer = ? AND oidc_subject = ?", issuer, subject).Take(&bySub).Error
	if err == nil {
		updates := map[string]any{"updated_at": time.Now().UTC()}
		if email != "" && !strings.EqualFold(bySub.Email, email) {
			// Keep email stable unless empty; do not steal another account's email.
			var other store.User
			if e2 := db.Where("LOWER(email) = ? AND id <> ?", email, bySub.ID).Take(&other).Error; e2 == nil {
				return store.User{}, &oidcLinkError{msg: "oidc subject linked to a different email already in use"}
			}
			updates["email"] = email
			bySub.Email = email
		}
		if name := strings.TrimSpace(claims.Name); name != "" && bySub.DisplayName == "" {
			updates["display_name"] = name
			bySub.DisplayName = name
		}
		_ = db.Model(&store.User{}).Where("id = ?", bySub.ID).Updates(updates).Error
		return bySub, nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return store.User{}, err
	}

	byEmail, err := h.userByLogin(c, email)
	if err == nil {
		if byEmail.OidcSubject != "" && (byEmail.OidcIssuer != issuer || byEmail.OidcSubject != subject) {
			return store.User{}, &oidcLinkError{
				msg: fmt.Sprintf("email %s already linked to another oidc subject", email),
			}
		}
		now := time.Now().UTC()
		if err := db.Model(&store.User{}).Where("id = ?", byEmail.ID).Updates(map[string]any{
			"oidc_issuer":  issuer,
			"oidc_subject": subject,
			"updated_at":   now,
		}).Error; err != nil {
			return store.User{}, err
		}
		byEmail.OidcIssuer = issuer
		byEmail.OidcSubject = subject
		byEmail.UpdatedAt = now
		return byEmail, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return store.User{}, err
	}

	now := time.Now().UTC()
	user := store.User{
		ID:          "user_" + uuid.NewString(),
		Email:       email,
		DisplayName: firstNonEmptyAPI(strings.TrimSpace(claims.Name), email),
		Status:      "active",
		OidcIssuer:  issuer,
		OidcSubject: subject,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := db.Create(&user).Error; err != nil {
		return store.User{}, err
	}
	return user, nil
}
