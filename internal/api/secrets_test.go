package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ash-repwiki/ash/internal/secrets"
	"github.com/ash-repwiki/ash/internal/store"
)

func TestSecretsAreSpaceScopedEncryptedAndAudited(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "jwt")
	t.Setenv("ASH_JWT_SECRET", "test-jwt-secret")
	t.Setenv("ASH_SECRET_KEY", "test-secret-key")
	r, db := newPlatformTestRouter(t)
	now := time.Now().UTC()
	org := store.Org{ID: "org_secrets", Name: "Secrets Org", Slug: "secrets-org", CreatedAt: now, UpdatedAt: now}
	ownSpace := store.Space{ID: "space_secrets_own", OrgID: org.ID, Name: "Own Secrets", Slug: "own-secrets", CreatedAt: now, UpdatedAt: now}
	otherSpace := store.Space{ID: "space_secrets_other", OrgID: org.ID, Name: "Other Secrets", Slug: "other-secrets", CreatedAt: now, UpdatedAt: now}
	for _, row := range []any{&org, &ownSpace, &otherSpace} {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	adminToken, err := signToken(tokenClaims{Sub: "secret-admin", SpaceID: ownSpace.ID, Role: "admin"}, "test-jwt-secret")
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/secrets", bytes.NewReader([]byte(`{
		"name":"OPENAI_API_KEY",
		"value":"sk-test-secret",
		"description":"model provider",
		"scope":{"runtime":"execgo","tool":"model-router"}
	}`)))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create status=%d want %d body=%s", w.Code, http.StatusCreated, w.Body.String())
	}
	var created SecretResponse
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.SpaceID != ownSpace.ID || created.Name != "OPENAI_API_KEY" || created.RedactedValue != "********" {
		t.Fatalf("created=%+v", created)
	}
	if strings.Contains(w.Body.String(), "sk-test-secret") {
		t.Fatalf("response leaked secret: %s", w.Body.String())
	}

	var row store.SecretRecord
	if err := db.First(&row, "id = ?", created.ID).Error; err != nil {
		t.Fatal(err)
	}
	if row.SpaceID != ownSpace.ID || row.ValueDigest != secrets.Digest("sk-test-secret") {
		t.Fatalf("row=%+v", row)
	}
	if strings.Contains(row.ValueCiphertext, "sk-test-secret") {
		t.Fatalf("ciphertext leaked plaintext: %s", row.ValueCiphertext)
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/secrets", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("list status=%d want %d body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "sk-test-secret") || strings.Contains(w.Body.String(), row.ValueCiphertext) {
		t.Fatalf("list leaked secret material: %s", w.Body.String())
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/secrets/"+created.ID+"/rotate", bytes.NewReader([]byte(`{"value":"sk-rotated-secret"}`)))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("rotate status=%d want %d body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	if err := db.First(&row, "id = ?", created.ID).Error; err != nil {
		t.Fatal(err)
	}
	if row.ValueDigest != secrets.Digest("sk-rotated-secret") {
		t.Fatalf("rotated digest=%q", row.ValueDigest)
	}

	otherToken, err := signToken(tokenClaims{Sub: "secret-admin", SpaceID: otherSpace.ID, Role: "admin"}, "test-jwt-secret")
	if err != nil {
		t.Fatal(err)
	}
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/secrets/"+created.ID+"/rotate", bytes.NewReader([]byte(`{"value":"nope"}`)))
	req.Header.Set("Authorization", "Bearer "+otherToken)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("cross-space rotate status=%d want %d body=%s", w.Code, http.StatusNotFound, w.Body.String())
	}

	var audits []store.AuditLog
	if err := db.Where("space_id = ? AND event_type IN ?", ownSpace.ID, []string{"secret.created", "secret.rotated"}).Find(&audits).Error; err != nil {
		t.Fatal(err)
	}
	if len(audits) != 2 {
		t.Fatalf("audits=%+v want created and rotated", audits)
	}
	for _, audit := range audits {
		if strings.Contains(audit.PayloadJSON, "sk-test-secret") || strings.Contains(audit.PayloadJSON, "sk-rotated-secret") {
			t.Fatalf("audit leaked secret: %+v", audit)
		}
	}
}

func TestSecretWriteRequiresPermission(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "jwt")
	t.Setenv("ASH_JWT_SECRET", "test-jwt-secret")
	r, _ := newPlatformTestRouter(t)
	token, err := signToken(tokenClaims{Sub: "secret-viewer", SpaceID: "local", Role: "viewer"}, "test-jwt-secret")
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/secrets", bytes.NewReader([]byte(`{"name":"TOKEN","value":"secret"}`)))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status=%d want %d body=%s", w.Code, http.StatusForbidden, w.Body.String())
	}
}
