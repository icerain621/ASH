package idp_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/ash-repwiki/ash/internal/idp"
)

func TestClientExchangeHS256(t *testing.T) {
	secret := "test-client-secret"
	mux := http.NewServeMux()
	var srv *httptest.Server
	srv = httptest.NewServer(mux)
	defer srv.Close()

	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{
			"issuer":                 srv.URL,
			"authorization_endpoint": srv.URL + "/authorize",
			"token_endpoint":         srv.URL + "/token",
		})
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if r.Form.Get("code") != "good-code" {
			http.Error(w, "bad code", http.StatusBadRequest)
			return
		}
		tok, err := idp.SignTestIDToken(secret, srv.URL, "client-1", "alice@example.com", "Alice", "sub-1", time.Now().Add(time.Hour))
		if err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"id_token": tok, "token_type": "Bearer"})
	})

	cli := idp.NewClient(idp.Config{
		Enabled: true, Issuer: srv.URL, ClientID: "client-1", ClientSecret: secret,
		RedirectURL: "http://localhost/cb", Scopes: "openid email",
		HTTPClient: srv.Client(),
	})
	authURL, state, err := cli.BeginLogin(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	if state == "" || authURL == "" {
		t.Fatalf("authURL=%q state=%q", authURL, state)
	}
	u, _ := url.Parse(authURL)
	if u.Query().Get("client_id") != "client-1" {
		t.Fatalf("authURL=%s", authURL)
	}
	ui, ok := cli.ConsumeState(state)
	if !ok || ui {
		t.Fatalf("state should be valid once without ui flag, ui=%v ok=%v", ui, ok)
	}
	if _, ok := cli.ConsumeState(state); ok {
		t.Fatal("state must be single-use")
	}

	claims, err := cli.ExchangeCode(context.Background(), "good-code")
	if err != nil {
		t.Fatal(err)
	}
	if claims.Email != "alice@example.com" || claims.Subject != "sub-1" || claims.Issuer != srv.URL {
		t.Fatalf("claims=%+v", claims)
	}
}

func TestClientExchangeRS256JWKS(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	kid := "test-key-1"
	mux := http.NewServeMux()
	var srv *httptest.Server
	srv = httptest.NewServer(mux)
	defer srv.Close()

	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{
			"issuer":                 srv.URL,
			"authorization_endpoint": srv.URL + "/authorize",
			"token_endpoint":         srv.URL + "/token",
			"jwks_uri":               srv.URL + "/jwks",
		})
	})
	mux.HandleFunc("/jwks", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"keys": []any{idp.PublicJWKFromRSA(&priv.PublicKey, kid)},
		})
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if r.Form.Get("code") != "rs-code" {
			http.Error(w, "bad code", http.StatusBadRequest)
			return
		}
		tok, err := idp.SignTestIDTokenRS256(priv, kid, srv.URL, "client-rs", "bob@example.com", "Bob", "sub-rs", time.Now().Add(time.Hour))
		if err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"id_token": tok})
	})

	cli := idp.NewClient(idp.Config{
		Enabled: true, Issuer: srv.URL, ClientID: "client-rs", ClientSecret: "unused-for-rs256",
		RedirectURL: "http://localhost/cb", Scopes: "openid email",
		HTTPClient: srv.Client(),
	})
	claims, err := cli.ExchangeCode(context.Background(), "rs-code")
	if err != nil {
		t.Fatal(err)
	}
	if claims.Email != "bob@example.com" || claims.Subject != "sub-rs" {
		t.Fatalf("claims=%+v", claims)
	}
}
