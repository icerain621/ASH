package idp

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

// Config holds optional OIDC settings (DX55). Empty / disabled when Enabled is false.
type Config struct {
	Enabled      bool
	Issuer       string
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       string
	DefaultSpace string
	HTTPClient   *http.Client
}

// Claims extracted from a validated id_token.
type Claims struct {
	Issuer  string
	Subject string
	Email   string
	Name    string
}

// LoadConfig reads ASH_OIDC_* env vars.
func LoadConfig() Config {
	enabled := envTruthy(os.Getenv("ASH_OIDC_ENABLED"))
	scopes := strings.TrimSpace(os.Getenv("ASH_OIDC_SCOPES"))
	if scopes == "" {
		scopes = "openid email profile"
	}
	return Config{
		Enabled:      enabled,
		Issuer:       strings.TrimRight(strings.TrimSpace(os.Getenv("ASH_OIDC_ISSUER")), "/"),
		ClientID:     strings.TrimSpace(os.Getenv("ASH_OIDC_CLIENT_ID")),
		ClientSecret: strings.TrimSpace(os.Getenv("ASH_OIDC_CLIENT_SECRET")),
		RedirectURL:  strings.TrimSpace(os.Getenv("ASH_OIDC_REDIRECT_URL")),
		Scopes:       scopes,
		DefaultSpace: strings.TrimSpace(os.Getenv("ASH_OIDC_DEFAULT_SPACE_ID")),
		HTTPClient:   http.DefaultClient,
	}
}

func (c Config) Valid() error {
	if !c.Enabled {
		return fmt.Errorf("oidc disabled")
	}
	if c.Issuer == "" || c.ClientID == "" || c.ClientSecret == "" || c.RedirectURL == "" {
		return fmt.Errorf("ASH_OIDC_ISSUER/CLIENT_ID/CLIENT_SECRET/REDIRECT_URL required when enabled")
	}
	return nil
}

type discoveryDoc struct {
	Issuer                string `json:"issuer"`
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
}

// Client performs OIDC authorization-code exchange (thin slice).
type Client struct {
	cfg   Config
	mu    sync.Mutex
	disco *discoveryDoc
	// states maps CSRF state → pending login metadata (process-local POC).
	states map[string]pendingLogin
}

type pendingLogin struct {
	Exp int64
	UI  bool
}

func NewClient(cfg Config) *Client {
	return &Client{cfg: cfg, states: map[string]pendingLogin{}}
}

func (c *Client) Enabled() bool {
	return c != nil && c.cfg.Enabled && c.cfg.Valid() == nil
}

func (c *Client) httpClient() *http.Client {
	if c.cfg.HTTPClient != nil {
		return c.cfg.HTTPClient
	}
	return http.DefaultClient
}

func (c *Client) discovery(ctx context.Context) (*discoveryDoc, error) {
	c.mu.Lock()
	if c.disco != nil {
		d := c.disco
		c.mu.Unlock()
		return d, nil
	}
	c.mu.Unlock()

	u := c.cfg.Issuer + "/.well-known/openid-configuration"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("oidc discovery status=%d body=%s", resp.StatusCode, string(body))
	}
	var doc discoveryDoc
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return nil, err
	}
	if doc.AuthorizationEndpoint == "" || doc.TokenEndpoint == "" {
		return nil, fmt.Errorf("oidc discovery missing endpoints")
	}
	if doc.Issuer == "" {
		doc.Issuer = c.cfg.Issuer
	}
	c.mu.Lock()
	c.disco = &doc
	c.mu.Unlock()
	return &doc, nil
}

// BeginLogin creates a state and returns the IdP authorize URL.
// When ui is true, ConsumeState will report UIRedirect so the API callback
// can 302 to the console login hash instead of returning JSON.
func (c *Client) BeginLogin(ctx context.Context, ui bool) (authURL, state string, err error) {
	if err := c.cfg.Valid(); err != nil {
		return "", "", err
	}
	doc, err := c.discovery(ctx)
	if err != nil {
		return "", "", err
	}
	state, err = randomState()
	if err != nil {
		return "", "", err
	}
	c.mu.Lock()
	c.pruneStatesLocked()
	c.states[state] = pendingLogin{Exp: time.Now().Add(10 * time.Minute).Unix(), UI: ui}
	c.mu.Unlock()

	q := url.Values{}
	q.Set("client_id", c.cfg.ClientID)
	q.Set("redirect_uri", c.cfg.RedirectURL)
	q.Set("response_type", "code")
	q.Set("scope", c.cfg.Scopes)
	q.Set("state", state)
	return doc.AuthorizationEndpoint + "?" + q.Encode(), state, nil
}

// ConsumeState validates and removes a CSRF state token.
// ok is false when missing/expired; ui mirrors BeginLogin's ui flag.
func (c *Client) ConsumeState(state string) (ui bool, ok bool) {
	state = strings.TrimSpace(state)
	if state == "" {
		return false, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.pruneStatesLocked()
	meta, found := c.states[state]
	if !found {
		return false, false
	}
	delete(c.states, state)
	if time.Now().Unix() > meta.Exp {
		return false, false
	}
	return meta.UI, true
}

func (c *Client) pruneStatesLocked() {
	now := time.Now().Unix()
	for k, meta := range c.states {
		if meta.Exp < now {
			delete(c.states, k)
		}
	}
}

type tokenResponse struct {
	IDToken     string `json:"id_token"`
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

// ExchangeCode trades an authorization code for validated claims.
func (c *Client) ExchangeCode(ctx context.Context, code string) (*Claims, error) {
	if err := c.cfg.Valid(); err != nil {
		return nil, err
	}
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, fmt.Errorf("code required")
	}
	doc, err := c.discovery(ctx)
	if err != nil {
		return nil, err
	}
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", c.cfg.RedirectURL)
	form.Set("client_id", c.cfg.ClientID)
	form.Set("client_secret", c.cfg.ClientSecret)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, doc.TokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("oidc token status=%d body=%s", resp.StatusCode, string(body))
	}
	var tok tokenResponse
	if err := json.Unmarshal(body, &tok); err != nil {
		return nil, err
	}
	if tok.IDToken == "" {
		return nil, fmt.Errorf("id_token missing")
	}
	return verifyIDTokenHS256(tok.IDToken, c.cfg.ClientSecret, c.cfg.ClientID, doc.Issuer)
}

type idTokenPayload struct {
	Iss   string `json:"iss"`
	Sub   string `json:"sub"`
	Aud   any    `json:"aud"`
	Exp   int64  `json:"exp"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

func verifyIDTokenHS256(raw, secret, clientID, issuer string) (*Claims, error) {
	parts := strings.Split(raw, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid id_token format")
	}
	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("id_token header: %w", err)
	}
	var header struct {
		Alg string `json:"alg"`
	}
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return nil, err
	}
	if !strings.EqualFold(header.Alg, "HS256") {
		return nil, fmt.Errorf("unsupported id_token alg %q (DX55 supports HS256; RS256 in later sprint)", header.Alg)
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(parts[0] + "." + parts[1]))
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, fmt.Errorf("id_token sig: %w", err)
	}
	if !hmac.Equal(sig, mac.Sum(nil)) {
		return nil, fmt.Errorf("id_token signature invalid")
	}
	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("id_token payload: %w", err)
	}
	var payload idTokenPayload
	if err := json.Unmarshal(payloadJSON, &payload); err != nil {
		return nil, err
	}
	if payload.Exp > 0 && time.Now().Unix() > payload.Exp {
		return nil, fmt.Errorf("id_token expired")
	}
	if issuer != "" && payload.Iss != "" && payload.Iss != issuer {
		return nil, fmt.Errorf("id_token iss mismatch")
	}
	if !audienceHas(payload.Aud, clientID) {
		return nil, fmt.Errorf("id_token aud mismatch")
	}
	email := strings.TrimSpace(payload.Email)
	if email == "" {
		return nil, fmt.Errorf("id_token email claim required")
	}
	name := strings.TrimSpace(payload.Name)
	if name == "" {
		name = email
	}
	sub := strings.TrimSpace(payload.Sub)
	if sub == "" {
		sub = email
	}
	iss := strings.TrimSpace(payload.Iss)
	if iss == "" {
		iss = strings.TrimSpace(issuer)
	}
	return &Claims{Issuer: iss, Subject: sub, Email: email, Name: name}, nil
}

func audienceHas(aud any, clientID string) bool {
	switch v := aud.(type) {
	case string:
		return v == clientID
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok && s == clientID {
				return true
			}
		}
	}
	return false
}

func randomState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func envTruthy(v string) bool {
	v = strings.TrimSpace(strings.ToLower(v))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

// SignTestIDToken builds an HS256 id_token for httptest mock IdPs (tests only).
func SignTestIDToken(secret, issuer, clientID, email, name, sub string, exp time.Time) (string, error) {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payloadBytes, err := json.Marshal(map[string]any{
		"iss": issuer, "aud": clientID, "sub": sub, "email": email, "name": name, "exp": exp.Unix(),
	})
	if err != nil {
		return "", err
	}
	payload := base64.RawURLEncoding.EncodeToString(payloadBytes)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(header + "." + payload))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return header + "." + payload + "." + sig, nil
}
