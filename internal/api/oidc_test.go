package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ash-repwiki/ash/internal/idp"
	"github.com/ash-repwiki/ash/internal/rules"
	"github.com/ash-repwiki/ash/internal/store"
)

func TestOIDCDisabledReturns404(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "dev")
	t.Setenv("ASH_OIDC_ENABLED", "0")
	r := newAuthTestRouter(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oidc/login", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestOIDCLoginCallbackIssuesJWT(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "jwt")
	t.Setenv("ASH_JWT_SECRET", "oidc-test-jwt-secret")

	secret := "oidc-client-secret"
	mux := http.NewServeMux()
	var idpSrv *httptest.Server
	idpSrv = httptest.NewServer(mux)
	defer idpSrv.Close()

	tokenEmail := "oidc.user@example.com"
	tokenSub := "idp-sub-9"
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{
			"issuer":                 idpSrv.URL,
			"authorization_endpoint": idpSrv.URL + "/authorize",
			"token_endpoint":         idpSrv.URL + "/token",
		})
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if r.Form.Get("code") != "auth-code-1" {
			http.Error(w, "bad code", 400)
			return
		}
		tok, err := idp.SignTestIDToken(secret, idpSrv.URL, "ash-client", tokenEmail, "OIDC User", tokenSub, time.Now().Add(time.Hour))
		if err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"id_token": tok})
	})

	db := store.OpenTest(t, t.TempDir())
	loader := rules.NewLoader(filepath.Join("..", "..", "scenarios"))
	if err := loader.LoadDir(); err != nil {
		t.Fatal(err)
	}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewHandler(db, loader)
	h.oidc = idp.NewClient(idp.Config{
		Enabled: true, Issuer: idpSrv.URL, ClientID: "ash-client", ClientSecret: secret,
		RedirectURL: "http://ash.test/api/v1/auth/oidc/callback",
		HTTPClient:  idpSrv.Client(),
	})
	h.Register(r, "")

	loginW := httptest.NewRecorder()
	loginReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oidc/login", nil)
	r.ServeHTTP(loginW, loginReq)
	if loginW.Code != http.StatusFound {
		t.Fatalf("login status=%d body=%s", loginW.Code, loginW.Body.String())
	}
	loc := loginW.Header().Get("Location")
	u, err := url.Parse(loc)
	if err != nil {
		t.Fatal(err)
	}
	state := u.Query().Get("state")
	if state == "" {
		t.Fatalf("missing state in %s", loc)
	}

	cbW := httptest.NewRecorder()
	cbReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oidc/callback?code=auth-code-1&state="+url.QueryEscape(state), nil)
	r.ServeHTTP(cbW, cbReq)
	if cbW.Code != http.StatusOK {
		t.Fatalf("callback status=%d body=%s", cbW.Code, cbW.Body.String())
	}
	var sess AuthSessionResponse
	if err := json.Unmarshal(cbW.Body.Bytes(), &sess); err != nil {
		t.Fatal(err)
	}
	if sess.Token == "" || sess.User.Email != "oidc.user@example.com" {
		t.Fatalf("sess=%+v", sess)
	}
	var user store.User
	if err := db.Where("LOWER(email) = ?", "oidc.user@example.com").First(&user).Error; err != nil {
		t.Fatalf("JIT user missing: %v", err)
	}
	if user.PasswordHash != "" {
		t.Fatalf("OIDC JIT user should have empty password hash")
	}
	if user.OidcIssuer != idpSrv.URL || user.OidcSubject != "idp-sub-9" {
		t.Fatalf("oidc link missing: issuer=%q subject=%q", user.OidcIssuer, user.OidcSubject)
	}

	// Second login: same subject, different email → reuse user via oidc link.
	tokenEmail = "oidc.user+alias@example.com"
	loginW2 := httptest.NewRecorder()
	r.ServeHTTP(loginW2, httptest.NewRequest(http.MethodGet, "/api/v1/auth/oidc/login", nil))
	state2, _ := url.Parse(loginW2.Header().Get("Location"))
	cbW2 := httptest.NewRecorder()
	r.ServeHTTP(cbW2, httptest.NewRequest(http.MethodGet, "/api/v1/auth/oidc/callback?code=auth-code-1&state="+url.QueryEscape(state2.Query().Get("state")), nil))
	if cbW2.Code != http.StatusOK {
		t.Fatalf("second callback status=%d body=%s", cbW2.Code, cbW2.Body.String())
	}
	var sess2 AuthSessionResponse
	_ = json.Unmarshal(cbW2.Body.Bytes(), &sess2)
	if sess2.User.ID != user.ID {
		t.Fatalf("want same user %s got %s", user.ID, sess2.User.ID)
	}
}

func TestOIDCEmailLinkAndConflict(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "jwt")
	t.Setenv("ASH_JWT_SECRET", "oidc-link-secret")
	secret := "link-secret"
	mux := http.NewServeMux()
	var idpSrv *httptest.Server
	idpSrv = httptest.NewServer(mux)
	defer idpSrv.Close()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{
			"issuer": idpSrv.URL, "authorization_endpoint": idpSrv.URL + "/authorize", "token_endpoint": idpSrv.URL + "/token",
		})
	})
	codeSub := "sub-a"
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		tok, err := idp.SignTestIDToken(secret, idpSrv.URL, "ash-client", "linkme@example.com", "Link Me", codeSub, time.Now().Add(time.Hour))
		if err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"id_token": tok})
	})

	db := store.OpenTest(t, t.TempDir())
	now := time.Now().UTC()
	existing := store.User{
		ID: "user_existing_link", Email: "linkme@example.com", DisplayName: "Local",
		PasswordHash: "x", Status: "active", CreatedAt: now, UpdatedAt: now,
	}
	if err := db.Create(&existing).Error; err != nil {
		t.Fatal(err)
	}
	loader := rules.NewLoader(filepath.Join("..", "..", "scenarios"))
	_ = loader.LoadDir()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewHandler(db, loader)
	h.oidc = idp.NewClient(idp.Config{
		Enabled: true, Issuer: idpSrv.URL, ClientID: "ash-client", ClientSecret: secret,
		RedirectURL: "http://ash.test/cb", HTTPClient: idpSrv.Client(),
	})
	h.Register(r, "")

	loginW := httptest.NewRecorder()
	r.ServeHTTP(loginW, httptest.NewRequest(http.MethodGet, "/api/v1/auth/oidc/login", nil))
	state, _ := url.Parse(loginW.Header().Get("Location"))
	cbW := httptest.NewRecorder()
	r.ServeHTTP(cbW, httptest.NewRequest(http.MethodGet, "/api/v1/auth/oidc/callback?code=c1&state="+url.QueryEscape(state.Query().Get("state")), nil))
	if cbW.Code != http.StatusOK {
		t.Fatalf("link status=%d body=%s", cbW.Code, cbW.Body.String())
	}
	var sess AuthSessionResponse
	_ = json.Unmarshal(cbW.Body.Bytes(), &sess)
	if sess.User.ID != existing.ID {
		t.Fatalf("want linked existing user, got %s", sess.User.ID)
	}
	var linked store.User
	_ = db.First(&linked, "id = ?", existing.ID).Error
	if linked.OidcSubject != "sub-a" {
		t.Fatalf("backfill subject=%q", linked.OidcSubject)
	}

	// Conflict: same email already linked to sub-a, try sub-b
	codeSub = "sub-b"
	loginW2 := httptest.NewRecorder()
	r.ServeHTTP(loginW2, httptest.NewRequest(http.MethodGet, "/api/v1/auth/oidc/login", nil))
	state2, _ := url.Parse(loginW2.Header().Get("Location"))
	cbW2 := httptest.NewRecorder()
	r.ServeHTTP(cbW2, httptest.NewRequest(http.MethodGet, "/api/v1/auth/oidc/callback?code=c2&state="+url.QueryEscape(state2.Query().Get("state")), nil))
	if cbW2.Code != http.StatusConflict {
		t.Fatalf("conflict status=%d body=%s", cbW2.Code, cbW2.Body.String())
	}
}

func TestOIDCDefaultSpaceEnv(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "dev")
	t.Setenv("ASH_JWT_SECRET", "oidc-space-secret")
	t.Setenv("ASH_OIDC_DEFAULT_SPACE_ID", "local")
	secret := "space-secret"
	mux := http.NewServeMux()
	var idpSrv *httptest.Server
	idpSrv = httptest.NewServer(mux)
	defer idpSrv.Close()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{
			"issuer": idpSrv.URL, "authorization_endpoint": idpSrv.URL + "/authorize", "token_endpoint": idpSrv.URL + "/token",
		})
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, _ *http.Request) {
		tok, _ := idp.SignTestIDToken(secret, idpSrv.URL, "ash-client", "space@example.com", "S", "sub-space", time.Now().Add(time.Hour))
		_ = json.NewEncoder(w).Encode(map[string]string{"id_token": tok})
	})
	db := store.OpenTest(t, t.TempDir())
	loader := rules.NewLoader(filepath.Join("..", "..", "scenarios"))
	_ = loader.LoadDir()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewHandler(db, loader)
	h.oidc = idp.NewClient(idp.Config{
		Enabled: true, Issuer: idpSrv.URL, ClientID: "ash-client", ClientSecret: secret,
		RedirectURL: "http://ash.test/cb", DefaultSpace: "local", HTTPClient: idpSrv.Client(),
	})
	h.Register(r, "")
	loginW := httptest.NewRecorder()
	r.ServeHTTP(loginW, httptest.NewRequest(http.MethodGet, "/api/v1/auth/oidc/login", nil))
	state, _ := url.Parse(loginW.Header().Get("Location"))
	cbW := httptest.NewRecorder()
	r.ServeHTTP(cbW, httptest.NewRequest(http.MethodGet, "/api/v1/auth/oidc/callback?code=c&state="+url.QueryEscape(state.Query().Get("state")), nil))
	if cbW.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", cbW.Code, cbW.Body.String())
	}
	var sess AuthSessionResponse
	_ = json.Unmarshal(cbW.Body.Bytes(), &sess)
	if sess.Space.ID != "local" {
		t.Fatalf("space=%+v", sess.Space)
	}
}

func TestOIDCUIRedirectToLoginHash(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "jwt")
	t.Setenv("ASH_JWT_SECRET", "oidc-ui-secret")
	secret := "ui-secret"
	mux := http.NewServeMux()
	var idpSrv *httptest.Server
	idpSrv = httptest.NewServer(mux)
	defer idpSrv.Close()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{
			"issuer": idpSrv.URL, "authorization_endpoint": idpSrv.URL + "/authorize", "token_endpoint": idpSrv.URL + "/token",
		})
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, _ *http.Request) {
		tok, _ := idp.SignTestIDToken(secret, idpSrv.URL, "ash-client", "ui@example.com", "UI", "sub-ui", time.Now().Add(time.Hour))
		_ = json.NewEncoder(w).Encode(map[string]string{"id_token": tok})
	})
	db := store.OpenTest(t, t.TempDir())
	loader := rules.NewLoader(filepath.Join("..", "..", "scenarios"))
	_ = loader.LoadDir()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewHandler(db, loader)
	h.oidc = idp.NewClient(idp.Config{
		Enabled: true, Issuer: idpSrv.URL, ClientID: "ash-client", ClientSecret: secret,
		RedirectURL: "http://ash.test/cb", HTTPClient: idpSrv.Client(),
	})
	h.Register(r, "")

	loginW := httptest.NewRecorder()
	r.ServeHTTP(loginW, httptest.NewRequest(http.MethodGet, "/api/v1/auth/oidc/login?ui=1", nil))
	if loginW.Code != http.StatusFound {
		t.Fatalf("login status=%d", loginW.Code)
	}
	state, _ := url.Parse(loginW.Header().Get("Location"))
	cbW := httptest.NewRecorder()
	r.ServeHTTP(cbW, httptest.NewRequest(http.MethodGet, "/api/v1/auth/oidc/callback?code=c&state="+url.QueryEscape(state.Query().Get("state")), nil))
	if cbW.Code != http.StatusFound {
		t.Fatalf("callback status=%d body=%s", cbW.Code, cbW.Body.String())
	}
	loc := cbW.Header().Get("Location")
	if !strings.HasPrefix(loc, "/ui/login#") || !strings.Contains(loc, "token=") {
		t.Fatalf("location=%s", loc)
	}
}
