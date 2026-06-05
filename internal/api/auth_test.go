package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ash-repwiki/ash/internal/rules"
	"github.com/ash-repwiki/ash/internal/store"
)

func newAuthTestRouter(t *testing.T) *gin.Engine {
	t.Helper()
	r, _ := newAuthTestRouterWithDB(t)
	return r
}

func newAuthTestRouterWithDB(t *testing.T) (*gin.Engine, *store.DB) {
	t.Helper()
	db := store.OpenTest(t, t.TempDir())
	scenariosDir := filepath.Join("..", "..", "scenarios")
	loader := rules.NewLoader(scenariosDir)
	if err := loader.LoadDir(); err != nil {
		t.Fatal(err)
	}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	NewHandler(db, loader).Register(r, "")
	return r, db
}

func TestJWTModeRequiresBearerToken(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "jwt")
	t.Setenv("ASH_JWT_SECRET", "test-secret")
	r := newAuthTestRouter(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/runs", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want %d body=%s", w.Code, http.StatusUnauthorized, w.Body.String())
	}
}

func TestDevLoginTokenAccess(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "jwt")
	t.Setenv("ASH_JWT_SECRET", "test-secret")
	r := newAuthTestRouter(t)

	login := httptest.NewRecorder()
	r.ServeHTTP(login, httptest.NewRequest(http.MethodPost, "/api/v1/auth/dev-login", nil))
	if login.Code != http.StatusOK {
		t.Fatalf("login status=%d body=%s", login.Code, login.Body.String())
	}

	token := extractTokenForTest(t, login.Body.String())
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/runs", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d want %d body=%s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestPasswordLoginIssuesSpaceScopedJWT(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "jwt")
	t.Setenv("ASH_JWT_SECRET", "test-secret")
	r, db := newAuthTestRouterWithDB(t)
	now := time.Now().UTC()
	org := store.Org{ID: "org_login", Name: "Login Org", Slug: "login-org", CreatedAt: now, UpdatedAt: now}
	space := store.Space{ID: "space_login", OrgID: org.ID, Name: "Login Space", Slug: "login-space", CreatedAt: now, UpdatedAt: now}
	passwordHash, err := hashPassword("correct-password")
	if err != nil {
		t.Fatal(err)
	}
	user := store.User{
		ID: "user_login", Email: "login@example.com", DisplayName: "Login User",
		PasswordHash: passwordHash, Status: "active", CreatedAt: now, UpdatedAt: now,
	}
	role := store.Role{
		ID: "role_login_mcp", OrgID: org.ID, Name: "tool-admin",
		Permissions: `["mcp:write"]`, CreatedAt: now, UpdatedAt: now,
	}
	member := store.Member{
		ID: "mem_login", OrgID: org.ID, SpaceID: space.ID, UserID: user.ID, RoleID: role.ID,
		Status: "active", CreatedAt: now, UpdatedAt: now,
	}
	for _, row := range []any{&org, &space, &user, &role, &member} {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}

	loginBody := []byte(`{"email":"login@example.com","password":"correct-password","spaceId":"` + space.ID + `"}`)
	login := httptest.NewRecorder()
	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(login, loginReq)
	if login.Code != http.StatusOK {
		t.Fatalf("login status=%d want %d body=%s", login.Code, http.StatusOK, login.Body.String())
	}
	token := extractTokenForTest(t, login.Body.String())
	claims, err := verifyToken(token, "test-secret")
	if err != nil {
		t.Fatal(err)
	}
	if claims.Sub != user.ID || claims.SpaceID != space.ID || claims.Role != "viewer" {
		t.Fatalf("claims=%+v want scoped viewer token", claims)
	}

	toolBody := []byte(`{"name":"repo.evidence","server":"http://127.0.0.1:7331"}`)
	tool := httptest.NewRecorder()
	toolReq := httptest.NewRequest(http.MethodPost, "/api/v1/mcp/tools", bytes.NewReader(toolBody))
	toolReq.Header.Set("Authorization", "Bearer "+token)
	toolReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(tool, toolReq)
	if tool.Code != http.StatusCreated {
		t.Fatalf("tool status=%d want %d body=%s", tool.Code, http.StatusCreated, tool.Body.String())
	}

	var audits []store.AuditLog
	if err := db.Where("space_id = ? AND actor_id = ? AND event_type = ?", space.ID, user.ID, "auth.login").Find(&audits).Error; err != nil {
		t.Fatal(err)
	}
	if len(audits) != 1 {
		t.Fatalf("auth login audits=%d want 1", len(audits))
	}
}

func TestPasswordLoginRejectsBadPasswordAndUnauthorizedSpace(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "jwt")
	t.Setenv("ASH_JWT_SECRET", "test-secret")
	r, db := newAuthTestRouterWithDB(t)
	now := time.Now().UTC()
	org := store.Org{ID: "org_login_reject", Name: "Reject Org", Slug: "reject-org", CreatedAt: now, UpdatedAt: now}
	ownSpace := store.Space{ID: "space_login_own", OrgID: org.ID, Name: "Own Login", Slug: "own-login", CreatedAt: now, UpdatedAt: now}
	otherSpace := store.Space{ID: "space_login_other", OrgID: org.ID, Name: "Other Login", Slug: "other-login", CreatedAt: now, UpdatedAt: now}
	passwordHash, err := hashPassword("correct-password")
	if err != nil {
		t.Fatal(err)
	}
	user := store.User{
		ID: "user_login_reject", Email: "reject@example.com", DisplayName: "Reject User",
		PasswordHash: passwordHash, Status: "active", CreatedAt: now, UpdatedAt: now,
	}
	role := store.Role{ID: "role_login_reject", OrgID: org.ID, Name: "viewer", Permissions: `["run:create"]`, CreatedAt: now, UpdatedAt: now}
	member := store.Member{
		ID: "mem_login_reject", OrgID: org.ID, SpaceID: ownSpace.ID, UserID: user.ID, RoleID: role.ID,
		Status: "active", CreatedAt: now, UpdatedAt: now,
	}
	for _, row := range []any{&org, &ownSpace, &otherSpace, &user, &role, &member} {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}

	bad := httptest.NewRecorder()
	badReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader([]byte(`{"email":"reject@example.com","password":"wrong","spaceId":"`+ownSpace.ID+`"}`)))
	badReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(bad, badReq)
	if bad.Code != http.StatusUnauthorized {
		t.Fatalf("bad password status=%d want %d body=%s", bad.Code, http.StatusUnauthorized, bad.Body.String())
	}

	other := httptest.NewRecorder()
	otherReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader([]byte(`{"email":"reject@example.com","password":"correct-password","spaceId":"`+otherSpace.ID+`"}`)))
	otherReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(other, otherReq)
	if other.Code != http.StatusForbidden {
		t.Fatalf("other space status=%d want %d body=%s", other.Code, http.StatusForbidden, other.Body.String())
	}
}

func TestChangePasswordRequiresAuthAndRotatesHash(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "jwt")
	t.Setenv("ASH_JWT_SECRET", "test-secret")
	r, db := newAuthTestRouterWithDB(t)
	now := time.Now().UTC()
	org := store.Org{ID: "org_change_password", Name: "Password Org", Slug: "password-org", CreatedAt: now, UpdatedAt: now}
	space := store.Space{ID: "space_change_password", OrgID: org.ID, Name: "Password Space", Slug: "password-space", CreatedAt: now, UpdatedAt: now}
	oldHash, err := hashPassword("old-password")
	if err != nil {
		t.Fatal(err)
	}
	user := store.User{
		ID: "user_change_password", Email: "change@example.com", DisplayName: "Change User",
		PasswordHash: oldHash, Status: "active", CreatedAt: now, UpdatedAt: now,
	}
	role := store.Role{ID: "role_change_password", OrgID: org.ID, Name: "viewer", Permissions: `["run:create"]`, CreatedAt: now, UpdatedAt: now}
	member := store.Member{
		ID: "mem_change_password", OrgID: org.ID, SpaceID: space.ID, UserID: user.ID, RoleID: role.ID,
		Status: "active", CreatedAt: now, UpdatedAt: now,
	}
	for _, row := range []any{&org, &space, &user, &role, &member} {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}

	unauth := httptest.NewRecorder()
	unauthReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/password", bytes.NewReader([]byte(`{"currentPassword":"old-password","newPassword":"new-password"}`)))
	unauthReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(unauth, unauthReq)
	if unauth.Code != http.StatusUnauthorized {
		t.Fatalf("unauth status=%d want %d body=%s", unauth.Code, http.StatusUnauthorized, unauth.Body.String())
	}

	token, err := signToken(tokenClaims{Sub: user.ID, SpaceID: space.ID, Role: "viewer"}, "test-secret")
	if err != nil {
		t.Fatal(err)
	}
	change := httptest.NewRecorder()
	changeReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/password", bytes.NewReader([]byte(`{"currentPassword":"old-password","newPassword":"new-password"}`)))
	changeReq.Header.Set("Authorization", "Bearer "+token)
	changeReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(change, changeReq)
	if change.Code != http.StatusOK {
		t.Fatalf("change status=%d want %d body=%s", change.Code, http.StatusOK, change.Body.String())
	}
	var updated store.User
	if err := db.First(&updated, "id = ?", user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if updated.PasswordHash == oldHash || !checkPasswordHash("new-password", updated.PasswordHash) {
		t.Fatalf("password hash was not rotated")
	}

	oldLogin := httptest.NewRecorder()
	oldLoginReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader([]byte(`{"email":"change@example.com","password":"old-password","spaceId":"`+space.ID+`"}`)))
	oldLoginReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(oldLogin, oldLoginReq)
	if oldLogin.Code != http.StatusUnauthorized {
		t.Fatalf("old login status=%d want %d body=%s", oldLogin.Code, http.StatusUnauthorized, oldLogin.Body.String())
	}

	newLogin := httptest.NewRecorder()
	newLoginReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader([]byte(`{"email":"change@example.com","password":"new-password","spaceId":"`+space.ID+`"}`)))
	newLoginReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(newLogin, newLoginReq)
	if newLogin.Code != http.StatusOK {
		t.Fatalf("new login status=%d want %d body=%s", newLogin.Code, http.StatusOK, newLogin.Body.String())
	}

	var audits []store.AuditLog
	if err := db.Where("space_id = ? AND actor_id = ? AND event_type = ?", space.ID, user.ID, "auth.password_changed").Find(&audits).Error; err != nil {
		t.Fatal(err)
	}
	if len(audits) != 1 {
		t.Fatalf("password change audits=%d want 1", len(audits))
	}
}

func TestAuthMeReturnsSpaceAndMemberPermissions(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "jwt")
	t.Setenv("ASH_JWT_SECRET", "test-secret")
	r, db := newAuthTestRouterWithDB(t)
	now := time.Now().UTC()
	org := store.Org{ID: "org_auth_me", Name: "Me Org", Slug: "me-org", CreatedAt: now, UpdatedAt: now}
	space := store.Space{ID: "space_auth_me", OrgID: org.ID, Name: "Me Space", Slug: "me-space", CreatedAt: now, UpdatedAt: now}
	user := store.User{ID: "user_auth_me", Email: "me@example.com", DisplayName: "Me User", Status: "active", CreatedAt: now, UpdatedAt: now}
	role := store.Role{
		ID: "role_auth_me", OrgID: org.ID, Name: "operator",
		Permissions: `["run:create","artifact:read","run:create"]`, CreatedAt: now, UpdatedAt: now,
	}
	member := store.Member{
		ID: "mem_auth_me", OrgID: org.ID, SpaceID: space.ID, UserID: user.ID, RoleID: role.ID,
		Status: "active", CreatedAt: now, UpdatedAt: now,
	}
	for _, row := range []any{&org, &space, &user, &role, &member} {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	token, err := signToken(tokenClaims{Sub: user.ID, SpaceID: space.ID, Role: "viewer"}, "test-secret")
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d want %d body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	var resp AuthMeResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.User.ID != user.ID || resp.User.Email != user.Email || resp.Space.ID != space.ID || resp.Space.Name != space.Name {
		t.Fatalf("resp=%+v want user and space", resp)
	}
	if len(resp.Permissions) != 2 || resp.Permissions[0] != "run:create" || resp.Permissions[1] != "artifact:read" {
		t.Fatalf("permissions=%+v want deduped member permissions", resp.Permissions)
	}
}

func TestAuthMeDevModeReturnsDefaultIdentity(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "dev")
	r := newAuthTestRouter(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d want %d body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	var resp AuthMeResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.User.ID != "dev-user" || resp.Space.ID != "local" || resp.Role != "admin" || len(resp.Permissions) != 1 || resp.Permissions[0] != "*" {
		t.Fatalf("resp=%+v want dev admin local", resp)
	}
}

func extractTokenForTest(t *testing.T, body string) string {
	t.Helper()
	var payload struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Token == "" {
		t.Fatal("empty token")
	}
	return payload.Token
}
