package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ash-repwiki/ash/internal/pluginabi"
	"github.com/ash-repwiki/ash/internal/store"
)

func TestRegisterPluginRequiresValidSignatureWhenKeySet(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "jwt")
	t.Setenv("ASH_JWT_SECRET", "test-secret")
	t.Setenv("ASH_PLUGIN_SIGNING_KEY", "unit-test-plugin-key")
	t.Setenv("ASH_PLUGIN_SIGNING_REQUIRED", "")
	r, db := newPlatformTestRouter(t)
	now := time.Now().UTC()
	org := store.Org{ID: "org_plugin_sign", Name: "Sign Org", Slug: "sign-org", CreatedAt: now, UpdatedAt: now}
	space := store.Space{ID: "space_plugin_sign", OrgID: org.ID, Name: "Sign", Slug: "sign", CreatedAt: now, UpdatedAt: now}
	user := store.User{ID: "user_plugin_sign", DisplayName: "Signer", Status: "active", CreatedAt: now, UpdatedAt: now}
	role := store.Role{ID: "role_plugin_sign", OrgID: org.ID, Name: "plugin", Permissions: `["plugin:*"]`, CreatedAt: now, UpdatedAt: now}
	member := store.Member{
		ID: "mem_plugin_sign", OrgID: org.ID, SpaceID: space.ID, UserID: user.ID, RoleID: role.ID,
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

	missing := httptest.NewRecorder()
	missingReq := httptest.NewRequest(http.MethodPost, "/api/v1/plugins", bytes.NewReader([]byte(
		`{"name":"otel","version":"1.0.0","protocol":"grpc","abi":"ash.plugin.v1","endpoint":"127.0.0.1:7443"}`,
	)))
	missingReq.Header.Set("Authorization", "Bearer "+token)
	missingReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(missing, missingReq)
	if missing.Code != http.StatusBadRequest {
		t.Fatalf("missing sig status=%d want 400 body=%s", missing.Code, missing.Body.String())
	}

	sig := pluginabi.SignHMAC("unit-test-plugin-key", "otel", "1.0.0", "grpc", "ash.plugin.v1", "127.0.0.1:7443")
	body, _ := json.Marshal(map[string]any{
		"name": "otel", "version": "1.0.0", "protocol": "grpc", "abi": "ash.plugin.v1",
		"endpoint": "127.0.0.1:7443", "signature": sig,
	})
	okResp := httptest.NewRecorder()
	okReq := httptest.NewRequest(http.MethodPost, "/api/v1/plugins", bytes.NewReader(body))
	okReq.Header.Set("Authorization", "Bearer "+token)
	okReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(okResp, okReq)
	if okResp.Code != http.StatusCreated {
		t.Fatalf("signed status=%d want 201 body=%s", okResp.Code, okResp.Body.String())
	}

	abiResp := httptest.NewRecorder()
	abiReq := httptest.NewRequest(http.MethodGet, "/api/v1/plugins/abi", nil)
	abiReq.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(abiResp, abiReq)
	if abiResp.Code != http.StatusOK {
		t.Fatalf("abi status=%d body=%s", abiResp.Code, abiResp.Body.String())
	}
	var profile PluginABIProfileResponse
	if err := json.Unmarshal(abiResp.Body.Bytes(), &profile); err != nil {
		t.Fatal(err)
	}
	if !profile.SigningRequired || !profile.SigningKeyConfigured || profile.SigningAlg != pluginabi.SignAlgHMAC {
		t.Fatalf("profile=%+v want signing required+configured hmac", profile)
	}
}
