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
	if primary.RefreshToken == "" {
		t.Fatal("expected refreshToken on login")
	}
	claims, err := verifyToken(primary.Token, "session-gateway-secret")
	if err != nil || claims.Sid != primary.Session.SID || claims.Typ != authSessionTypPrimary {
		t.Fatalf("claims=%+v err=%v", claims, err)
	}
	refreshClaims, err := verifyToken(primary.RefreshToken, "session-gateway-secret")
	if err != nil || refreshClaims.Typ != authSessionTypRefresh || refreshClaims.Sid != primary.Session.SID {
		t.Fatalf("refresh claims=%+v err=%v", refreshClaims, err)
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
	if primary.RefreshToken == "" {
		t.Fatal("expected refreshToken on login")
	}

	// Preferred path: Bearer refresh token
	ref := httptest.NewRecorder()
	refReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/sessions/refresh", bytes.NewReader([]byte(`{"ttlSeconds":3600}`)))
	refReq.Header.Set("Authorization", "Bearer "+primary.RefreshToken)
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
	if refreshed.RefreshToken == "" || refreshed.RefreshToken == primary.RefreshToken {
		t.Fatalf("refresh token should rotate: got=%q old=%q", refreshed.RefreshToken, primary.RefreshToken)
	}
	if refreshed.Token == "" || refreshed.Token == primary.Token {
		t.Fatalf("access token should rotate")
	}
	var row store.AuditLog
	if err := db.First(&row, "id = ?", primary.Session.SID).Error; err != nil {
		t.Fatal(err)
	}
	var payload authSessionPayload
	if err := json.Unmarshal([]byte(row.PayloadJSON), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.RotateCount < 1 || payload.RotatedAt == 0 || payload.RefreshExp == 0 {
		t.Fatalf("rotation fields=%+v", payload)
	}

	// Legacy path: unexpired access token still refreshes
	legacy := httptest.NewRecorder()
	legacyReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/sessions/refresh", bytes.NewReader([]byte(`{}`)))
	legacyReq.Header.Set("Authorization", "Bearer "+refreshed.Token)
	legacyReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(legacy, legacyReq)
	if legacy.Code != http.StatusOK {
		t.Fatalf("legacy refresh status=%d body=%s", legacy.Code, legacy.Body.String())
	}
	_ = json.Unmarshal(legacy.Body.Bytes(), &refreshed)

	me := httptest.NewRecorder()
	meReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	meReq.Header.Set("Authorization", "Bearer "+refreshed.Token)
	r.ServeHTTP(me, meReq)
	if me.Code != http.StatusOK {
		t.Fatalf("me after refresh status=%d body=%s", me.Code, me.Body.String())
	}

	misuse := httptest.NewRecorder()
	misuseReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	misuseReq.Header.Set("Authorization", "Bearer "+refreshed.RefreshToken)
	r.ServeHTTP(misuse, misuseReq)
	if misuse.Code != http.StatusUnauthorized || !strings.Contains(misuse.Body.String(), "AUTH_REFRESH_TOKEN_MISUSE") {
		t.Fatalf("refresh misuse status=%d body=%s", misuse.Code, misuse.Body.String())
	}

	rev := httptest.NewRecorder()
	revReq := httptest.NewRequest(http.MethodDelete, "/api/v1/auth/sessions/"+primary.Session.SID, nil)
	revReq.Header.Set("Authorization", "Bearer "+refreshed.Token)
	r.ServeHTTP(rev, revReq)
	if rev.Code != http.StatusOK {
		t.Fatalf("revoke status=%d body=%s", rev.Code, rev.Body.String())
	}
	for _, tok := range []string{primary.Token, refreshed.Token, refreshed.RefreshToken} {
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
	if expRef.Code != http.StatusUnauthorized || !strings.Contains(expRef.Body.String(), "AUTH_REFRESH_REQUIRED") {
		t.Fatalf("expired refresh status=%d body=%s", expRef.Code, expRef.Body.String())
	}

	// Body refreshToken without Authorization.
	bodyRef := httptest.NewRecorder()
	freshAccess, err := signToken(tokenClaims{
		Sub: user.ID, SpaceID: space.ID, Role: "viewer", Exp: time.Now().Add(time.Hour).Unix(),
		Sid: primary.Session.SID, Did: "did_login", Typ: authSessionTypPrimary,
	}, "session-refresh-secret")
	if err != nil {
		t.Fatal(err)
	}
	pair := httptest.NewRecorder()
	pairReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/sessions/refresh", bytes.NewReader([]byte(`{}`)))
	pairReq.Header.Set("Authorization", "Bearer "+freshAccess)
	pairReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(pair, pairReq)
	if pair.Code != http.StatusOK {
		t.Fatalf("reissue for body test status=%d body=%s", pair.Code, pair.Body.String())
	}
	var live AuthSessionResponse
	_ = json.Unmarshal(pair.Body.Bytes(), &live)
	bodyJSON, _ := json.Marshal(map[string]string{"refreshToken": live.RefreshToken})
	bodyReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/sessions/refresh", bytes.NewReader(bodyJSON))
	bodyReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(bodyRef, bodyReq)
	if bodyRef.Code != http.StatusOK {
		t.Fatalf("body refresh status=%d body=%s", bodyRef.Code, bodyRef.Body.String())
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

func TestAuthSessionJtiReplayAndGrace(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "jwt")
	t.Setenv("ASH_JWT_SECRET", "session-jti-secret")
	t.Setenv("ASH_AUTH_TOKEN_GRACE_SEC", "0")
	r, db := newAuthTestRouterWithDB(t)
	now := time.Now().UTC()
	org := store.Org{ID: "org_jti", Name: "Jti Org", Slug: "jti-org", CreatedAt: now, UpdatedAt: now}
	space := store.Space{ID: "space_jti", OrgID: org.ID, Name: "Jti Space", Slug: "jti-space", CreatedAt: now, UpdatedAt: now}
	passwordHash, err := hashPassword("correct-password")
	if err != nil {
		t.Fatal(err)
	}
	user := store.User{
		ID: "user_jti", Email: "jti@example.com", DisplayName: "Jti",
		PasswordHash: passwordHash, Status: "active", CreatedAt: now, UpdatedAt: now,
	}
	role := store.Role{ID: "role_jti", OrgID: org.ID, Name: "viewer", Permissions: `["run:create"]`, CreatedAt: now, UpdatedAt: now}
	member := store.Member{
		ID: "mem_jti", OrgID: org.ID, SpaceID: space.ID, UserID: user.ID, RoleID: role.ID,
		Status: "active", CreatedAt: now, UpdatedAt: now,
	}
	for _, row := range []any{&org, &space, &user, &role, &member} {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}

	login := httptest.NewRecorder()
	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader([]byte(
		`{"email":"jti@example.com","password":"correct-password","spaceId":"`+space.ID+`"}`,
	)))
	loginReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(login, loginReq)
	if login.Code != http.StatusOK {
		t.Fatalf("login status=%d body=%s", login.Code, login.Body.String())
	}
	var primary AuthSessionResponse
	_ = json.Unmarshal(login.Body.Bytes(), &primary)
	if primary.RefreshToken == "" || primary.Session == nil {
		t.Fatalf("login pair incomplete: %+v", primary)
	}
	var row store.AuditLog
	if err := db.First(&row, "id = ?", primary.Session.SID).Error; err != nil {
		t.Fatal(err)
	}
	var payload authSessionPayload
	_ = json.Unmarshal([]byte(row.PayloadJSON), &payload)
	if payload.AccessJti == "" || payload.RefreshJti == "" {
		t.Fatalf("expected jtis stored on issue: %+v", payload)
	}

	ref := httptest.NewRecorder()
	refReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/sessions/refresh", bytes.NewReader([]byte(`{}`)))
	refReq.Header.Set("Authorization", "Bearer "+primary.RefreshToken)
	refReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(ref, refReq)
	if ref.Code != http.StatusOK {
		t.Fatalf("refresh status=%d body=%s", ref.Code, ref.Body.String())
	}
	var refreshed AuthSessionResponse
	_ = json.Unmarshal(ref.Body.Bytes(), &refreshed)

	// grace=0: old access/refresh rejected
	oldMe := httptest.NewRecorder()
	oldMeReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	oldMeReq.Header.Set("Authorization", "Bearer "+primary.Token)
	r.ServeHTTP(oldMe, oldMeReq)
	if oldMe.Code != http.StatusUnauthorized || !strings.Contains(oldMe.Body.String(), "AUTH_TOKEN_REPLAY") {
		t.Fatalf("old access after rotate status=%d body=%s", oldMe.Code, oldMe.Body.String())
	}
	oldRef := httptest.NewRecorder()
	oldRefReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/sessions/refresh", bytes.NewReader([]byte(`{}`)))
	oldRefReq.Header.Set("Authorization", "Bearer "+primary.RefreshToken)
	oldRefReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(oldRef, oldRefReq)
	if oldRef.Code != http.StatusUnauthorized || !strings.Contains(oldRef.Body.String(), "AUTH_TOKEN_REPLAY") {
		t.Fatalf("old refresh replay status=%d body=%s", oldRef.Code, oldRef.Body.String())
	}

	// current tokens OK
	me := httptest.NewRecorder()
	meReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	meReq.Header.Set("Authorization", "Bearer "+refreshed.Token)
	r.ServeHTTP(me, meReq)
	if me.Code != http.StatusOK {
		t.Fatalf("current access status=%d body=%s", me.Code, me.Body.String())
	}

	// grace window accepts prev
	t.Setenv("ASH_AUTH_TOKEN_GRACE_SEC", "120")
	_ = db.First(&row, "id = ?", primary.Session.SID).Error
	_ = json.Unmarshal([]byte(row.PayloadJSON), &payload)
	payload.RotatedAt = time.Now().UTC().Unix()
	b, _ := json.Marshal(payload)
	_ = db.Model(&store.AuditLog{}).Where("id = ?", primary.Session.SID).Update("payload_json", string(b)).Error

	graceMe := httptest.NewRecorder()
	graceMeReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	graceMeReq.Header.Set("Authorization", "Bearer "+primary.Token)
	r.ServeHTTP(graceMe, graceMeReq)
	if graceMe.Code != http.StatusOK {
		t.Fatalf("prev access within grace status=%d body=%s", graceMe.Code, graceMe.Body.String())
	}

	// legacy no-jti still works while active
	legacyTok, err := signToken(tokenClaims{
		Sub: user.ID, SpaceID: space.ID, Role: "viewer", Exp: time.Now().Add(time.Hour).Unix(),
		Sid: primary.Session.SID, Did: "did_login", Typ: authSessionTypPrimary,
	}, "session-jti-secret")
	if err != nil {
		t.Fatal(err)
	}
	leg := httptest.NewRecorder()
	legReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	legReq.Header.Set("Authorization", "Bearer "+legacyTok)
	r.ServeHTTP(leg, legReq)
	if leg.Code != http.StatusOK {
		t.Fatalf("legacy no-jti status=%d body=%s", leg.Code, leg.Body.String())
	}

	rev := httptest.NewRecorder()
	revReq := httptest.NewRequest(http.MethodDelete, "/api/v1/auth/sessions/"+primary.Session.SID, nil)
	revReq.Header.Set("Authorization", "Bearer "+refreshed.Token)
	r.ServeHTTP(rev, revReq)
	if rev.Code != http.StatusOK {
		t.Fatalf("revoke status=%d body=%s", rev.Code, rev.Body.String())
	}
	denied := httptest.NewRecorder()
	deniedReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	deniedReq.Header.Set("Authorization", "Bearer "+refreshed.Token)
	r.ServeHTTP(denied, deniedReq)
	if denied.Code != http.StatusUnauthorized || !strings.Contains(denied.Body.String(), "AUTH_SESSION_REVOKED") {
		t.Fatalf("revoked current status=%d body=%s", denied.Code, denied.Body.String())
	}
}
