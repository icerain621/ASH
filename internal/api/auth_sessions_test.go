package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ash-repwiki/ash/internal/store"
)

func TestAuthSessionDeviceMintListAndRevoke(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "jwt")
	t.Setenv("ASH_JWT_SECRET", "session-gateway-secret")
	r, db := newAuthTestRouterWithDB(t)
	now := time.Now().UTC()
	org := store.Org{ID: "org_sess", Name: "Sess Org", Slug: "sess-org", CreatedAt: now, UpdatedAt: now}
	space := store.Space{ID: "space_sess", OrgID: org.ID, Name: "Sess Space", Slug: "sess-space", CreatedAt: now, UpdatedAt: now}
	passwordHash, err := hashPassword("correct-password")
	if err != nil {
		t.Fatal(err)
	}
	user := store.User{
		ID: "user_sess", Email: "sess@example.com", DisplayName: "Sess",
		PasswordHash: passwordHash, Status: "active", CreatedAt: now, UpdatedAt: now,
	}
	role := store.Role{ID: "role_sess", OrgID: org.ID, Name: "viewer", Permissions: `["run:create"]`, CreatedAt: now, UpdatedAt: now}
	member := store.Member{
		ID: "mem_sess", OrgID: org.ID, SpaceID: space.ID, UserID: user.ID, RoleID: role.ID,
		Status: "active", CreatedAt: now, UpdatedAt: now,
	}
	for _, row := range []any{&org, &space, &user, &role, &member} {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}

	login := httptest.NewRecorder()
	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader([]byte(
		`{"email":"sess@example.com","password":"correct-password","spaceId":"`+space.ID+`"}`,
	)))
	loginReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(login, loginReq)
	if login.Code != http.StatusOK {
		t.Fatalf("login status=%d body=%s", login.Code, login.Body.String())
	}
	var primary AuthSessionResponse
	if err := json.Unmarshal(login.Body.Bytes(), &primary); err != nil {
		t.Fatal(err)
	}
	if primary.Session == nil || primary.Session.SID == "" || primary.Session.Typ != authSessionTypPrimary {
		t.Fatalf("primary session=%+v", primary.Session)
	}
	claims, err := verifyToken(primary.Token, "session-gateway-secret")
	if err != nil || claims.Sid != primary.Session.SID || claims.Typ != authSessionTypPrimary {
		t.Fatalf("claims=%+v err=%v", claims, err)
	}

	dev := httptest.NewRecorder()
	devReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/sessions/device", bytes.NewReader([]byte(
		`{"deviceId":"laptop-1","ttlSeconds":3600}`,
	)))
	devReq.Header.Set("Authorization", "Bearer "+primary.Token)
	devReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(dev, devReq)
	if dev.Code != http.StatusOK {
		t.Fatalf("device status=%d body=%s", dev.Code, dev.Body.String())
	}
	var device AuthSessionResponse
	if err := json.Unmarshal(dev.Body.Bytes(), &device); err != nil {
		t.Fatal(err)
	}
	if device.Session == nil || device.Session.Typ != authSessionTypDevice || device.Session.DID != "laptop-1" {
		t.Fatalf("device session=%+v", device.Session)
	}

	list := httptest.NewRecorder()
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/sessions", nil)
	listReq.Header.Set("Authorization", "Bearer "+primary.Token)
	r.ServeHTTP(list, listReq)
	if list.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", list.Code, list.Body.String())
	}
	var listed authSessionListResponse
	_ = json.Unmarshal(list.Body.Bytes(), &listed)
	if len(listed.Items) < 2 {
		t.Fatalf("items=%d want >=2", len(listed.Items))
	}

	rev := httptest.NewRecorder()
	revReq := httptest.NewRequest(http.MethodDelete, "/api/v1/auth/sessions/"+device.Session.SID, nil)
	revReq.Header.Set("Authorization", "Bearer "+primary.Token)
	r.ServeHTTP(rev, revReq)
	if rev.Code != http.StatusOK {
		t.Fatalf("revoke status=%d body=%s", rev.Code, rev.Body.String())
	}

	denied := httptest.NewRecorder()
	deniedReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	deniedReq.Header.Set("Authorization", "Bearer "+device.Token)
	r.ServeHTTP(denied, deniedReq)
	if denied.Code != http.StatusUnauthorized {
		t.Fatalf("revoked device token status=%d body=%s", denied.Code, denied.Body.String())
	}
}

func TestAuthSessionRefreshAndRevoke(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "jwt")
	t.Setenv("ASH_JWT_SECRET", "session-refresh-secret")
	r, db := newAuthTestRouterWithDB(t)
	now := time.Now().UTC()
	org := store.Org{ID: "org_ref", Name: "Ref Org", Slug: "ref-org", CreatedAt: now, UpdatedAt: now}
	space := store.Space{ID: "space_ref", OrgID: org.ID, Name: "Ref Space", Slug: "ref-space", CreatedAt: now, UpdatedAt: now}
	passwordHash, err := hashPassword("correct-password")
	if err != nil {
		t.Fatal(err)
	}
	user := store.User{
		ID: "user_ref", Email: "ref@example.com", DisplayName: "Ref",
		PasswordHash: passwordHash, Status: "active", CreatedAt: now, UpdatedAt: now,
	}
	role := store.Role{ID: "role_ref", OrgID: org.ID, Name: "viewer", Permissions: `["run:create"]`, CreatedAt: now, UpdatedAt: now}
	member := store.Member{
		ID: "mem_ref", OrgID: org.ID, SpaceID: space.ID, UserID: user.ID, RoleID: role.ID,
		Status: "active", CreatedAt: now, UpdatedAt: now,
	}
	for _, row := range []any{&org, &space, &user, &role, &member} {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}

	login := httptest.NewRecorder()
	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader([]byte(
		`{"email":"ref@example.com","password":"correct-password","spaceId":"`+space.ID+`"}`,
	)))
	loginReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(login, loginReq)
	if login.Code != http.StatusOK {
		t.Fatalf("login status=%d body=%s", login.Code, login.Body.String())
	}
	var primary AuthSessionResponse
	if err := json.Unmarshal(login.Body.Bytes(), &primary); err != nil {
		t.Fatal(err)
	}

	ref := httptest.NewRecorder()
	refReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/sessions/refresh", bytes.NewReader([]byte(`{"ttlSeconds":3600}`)))
	refReq.Header.Set("Authorization", "Bearer "+primary.Token)
	refReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(ref, refReq)
	if ref.Code != http.StatusOK {
		t.Fatalf("refresh status=%d body=%s", ref.Code, ref.Body.String())
	}
	var refreshed AuthSessionResponse
	if err := json.Unmarshal(ref.Body.Bytes(), &refreshed); err != nil {
		t.Fatal(err)
	}
	if refreshed.Session == nil || refreshed.Session.SID != primary.Session.SID {
		t.Fatalf("sid changed: got=%+v want=%s", refreshed.Session, primary.Session.SID)
	}
	var row store.AuditLog
	if err := db.First(&row, "id = ?", primary.Session.SID).Error; err != nil {
		t.Fatal(err)
	}
	var payload authSessionPayload
	if err := json.Unmarshal([]byte(row.PayloadJSON), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.RotateCount < 1 || payload.RotatedAt == 0 {
		t.Fatalf("rotation fields=%+v", payload)
	}

	me := httptest.NewRecorder()
	meReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	meReq.Header.Set("Authorization", "Bearer "+refreshed.Token)
	r.ServeHTTP(me, meReq)
	if me.Code != http.StatusOK {
		t.Fatalf("me after refresh status=%d body=%s", me.Code, me.Body.String())
	}

	rev := httptest.NewRecorder()
	revReq := httptest.NewRequest(http.MethodDelete, "/api/v1/auth/sessions/"+primary.Session.SID, nil)
	revReq.Header.Set("Authorization", "Bearer "+refreshed.Token)
	r.ServeHTTP(rev, revReq)
	if rev.Code != http.StatusOK {
		t.Fatalf("revoke status=%d body=%s", rev.Code, rev.Body.String())
	}
	for _, tok := range []string{primary.Token, refreshed.Token} {
		denied := httptest.NewRecorder()
		deniedReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
		deniedReq.Header.Set("Authorization", "Bearer "+tok)
		r.ServeHTTP(denied, deniedReq)
		if denied.Code != http.StatusUnauthorized {
			t.Fatalf("revoked token status=%d body=%s", denied.Code, denied.Body.String())
		}
	}

	expiredTok, err := signToken(tokenClaims{
		Sub: user.ID, SpaceID: space.ID, Role: "viewer", Exp: time.Now().Add(-time.Hour).Unix(),
		Sid: primary.Session.SID, Did: "did_login", Typ: authSessionTypPrimary,
	}, "session-refresh-secret")
	if err != nil {
		t.Fatal(err)
	}
	// Re-activate registry for expired-token path (sid still exists after revoke).
	payload.Status = authSessionStatusActive
	b, _ := json.Marshal(payload)
	_ = db.Model(&store.AuditLog{}).Where("id = ?", primary.Session.SID).Update("payload_json", string(b)).Error

	expRef := httptest.NewRecorder()
	expReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/sessions/refresh", bytes.NewReader([]byte(`{}`)))
	expReq.Header.Set("Authorization", "Bearer "+expiredTok)
	expReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(expRef, expReq)
	if expRef.Code != http.StatusUnauthorized || !strings.Contains(expRef.Body.String(), "AUTH_SESSION_EXPIRED") {
		t.Fatalf("expired refresh status=%d body=%s", expRef.Code, expRef.Body.String())
	}
}

func TestAuthSessionScopeEnforcement(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "jwt")
	t.Setenv("ASH_JWT_SECRET", "session-scope-secret")
	r, db := newAuthTestRouterWithDB(t)
	now := time.Now().UTC()
	org := store.Org{ID: "org_scope", Name: "Scope Org", Slug: "scope-org", CreatedAt: now, UpdatedAt: now}
	space := store.Space{ID: "space_scope", OrgID: org.ID, Name: "Scope Space", Slug: "scope-space", CreatedAt: now, UpdatedAt: now}
	passwordHash, err := hashPassword("correct-password")
	if err != nil {
		t.Fatal(err)
	}
	user := store.User{
		ID: "user_scope", Email: "scope@example.com", DisplayName: "Scope",
		PasswordHash: passwordHash, Status: "active", CreatedAt: now, UpdatedAt: now,
	}
	role := store.Role{ID: "role_scope", OrgID: org.ID, Name: "viewer", Permissions: `["run:create"]`, CreatedAt: now, UpdatedAt: now}
	member := store.Member{
		ID: "mem_scope", OrgID: org.ID, SpaceID: space.ID, UserID: user.ID, RoleID: role.ID,
		Status: "active", CreatedAt: now, UpdatedAt: now,
	}
	for _, row := range []any{&org, &space, &user, &role, &member} {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}

	login := httptest.NewRecorder()
	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader([]byte(
		`{"email":"scope@example.com","password":"correct-password","spaceId":"`+space.ID+`"}`,
	)))
	loginReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(login, loginReq)
	if login.Code != http.StatusOK {
		t.Fatalf("login status=%d body=%s", login.Code, login.Body.String())
	}
	var primary AuthSessionResponse
	_ = json.Unmarshal(login.Body.Bytes(), &primary)

	badMint := httptest.NewRecorder()
	badReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/sessions/device", bytes.NewReader([]byte(
		`{"deviceId":"bad","scope":["mcp:write"]}`,
	)))
	badReq.Header.Set("Authorization", "Bearer "+primary.Token)
	badReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(badMint, badReq)
	if badMint.Code != http.StatusBadRequest || !strings.Contains(badMint.Body.String(), "AUTH_SCOPE_INVALID") {
		t.Fatalf("invalid scope mint status=%d body=%s", badMint.Code, badMint.Body.String())
	}

	narrow := httptest.NewRecorder()
	narrowReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/sessions/device", bytes.NewReader([]byte(
		`{"deviceId":"narrow","scope":["artifact:read"]}`,
	)))
	narrowReq.Header.Set("Authorization", "Bearer "+primary.Token)
	narrowReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(narrow, narrowReq)
	if narrow.Code != http.StatusOK {
		t.Fatalf("narrow mint status=%d body=%s", narrow.Code, narrow.Body.String())
	}
	var device AuthSessionResponse
	_ = json.Unmarshal(narrow.Body.Bytes(), &device)

	denied := httptest.NewRecorder()
	deniedReq := httptest.NewRequest(http.MethodPost, "/api/v1/runs", bytes.NewReader([]byte(
		`{"scenario":{"name":"feature_delivery","scenarioVersion":"1.0.0"},"inputs":{},"spaceId":"`+space.ID+`"}`,
	)))
	deniedReq.Header.Set("Authorization", "Bearer "+device.Token)
	deniedReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(denied, deniedReq)
	if denied.Code != http.StatusForbidden || !strings.Contains(denied.Body.String(), "AUTH_SCOPE_DENIED") {
		t.Fatalf("scope deny status=%d body=%s", denied.Code, denied.Body.String())
	}

	okMint := httptest.NewRecorder()
	okReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/sessions/device", bytes.NewReader([]byte(
		`{"deviceId":"ok","scope":["run:create"]}`,
	)))
	okReq.Header.Set("Authorization", "Bearer "+primary.Token)
	okReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(okMint, okReq)
	if okMint.Code != http.StatusOK {
		t.Fatalf("ok mint status=%d body=%s", okMint.Code, okMint.Body.String())
	}
	var allowDev AuthSessionResponse
	_ = json.Unmarshal(okMint.Body.Bytes(), &allowDev)
	allowed := httptest.NewRecorder()
	allowReq := httptest.NewRequest(http.MethodPost, "/api/v1/runs", bytes.NewReader([]byte(
		`{"scenario":{"name":"feature_delivery","scenarioVersion":"1.0.0"},"inputs":{},"spaceId":"`+space.ID+`"}`,
	)))
	allowReq.Header.Set("Authorization", "Bearer "+allowDev.Token)
	allowReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(allowed, allowReq)
	if allowed.Code == http.StatusForbidden {
		t.Fatalf("scoped run:create should pass permission gate, body=%s", allowed.Body.String())
	}
}
