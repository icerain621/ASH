package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ash-repwiki/ash/internal/store"
)

func TestCreateRepoConnectionRejectsPlaintextToken(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "dev")
	r, _ := newPlatformTestRouter(t)
	body := []byte(`{"owner":"iammm0","repo":"ASH","token":"plaintext"}`)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/repo/connections", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want 400 body=%s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("PLAINTEXT_TOKEN_REJECTED")) {
		t.Fatalf("body=%s want PLAINTEXT_TOKEN_REJECTED", w.Body.String())
	}
}

func TestRepoConnectionAndCIDiagnoseAPI(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "dev")
	r, db := newPlatformTestRouter(t)
	secretID := createGitHubSecret(t, r)

	connBody := []byte(`{"provider":"github","owner":"iammm0","repo":"ASH","secretId":"` + secretID + `"}`)
	connResp := httptest.NewRecorder()
	connReq := httptest.NewRequest(http.MethodPost, "/api/v1/repo/connections", bytes.NewReader(connBody))
	connReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(connResp, connReq)
	if connResp.Code != http.StatusCreated {
		t.Fatalf("status=%d want 201 body=%s", connResp.Code, connResp.Body.String())
	}
	var conn store.RepoConnection
	if err := json.Unmarshal(connResp.Body.Bytes(), &conn); err != nil {
		t.Fatal(err)
	}
	if conn.ID == "" || conn.SecretID != secretID {
		t.Fatalf("conn=%+v want id and secret", conn)
	}

	diagBody := []byte(`{"connectionId":"` + conn.ID + `","logText":"go test ./...\n--- FAIL: TestAPI (0.01s)\nFAIL\tpkg\t0.1s"}`)
	diagResp := httptest.NewRecorder()
	diagReq := httptest.NewRequest(http.MethodPost, "/api/v1/ci/failures/diagnose", bytes.NewReader(diagBody))
	diagReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(diagResp, diagReq)
	if diagResp.Code != http.StatusOK {
		t.Fatalf("status=%d want 200 body=%s", diagResp.Code, diagResp.Body.String())
	}
	var diag struct {
		RootCause string `json:"rootCause"`
	}
	if err := json.Unmarshal(diagResp.Body.Bytes(), &diag); err != nil {
		t.Fatal(err)
	}
	if diag.RootCause != "test_failure" {
		t.Fatalf("diag=%+v want test_failure", diag)
	}
	var count int64
	if err := db.Model(&store.CIDiagnosis{}).Where("connection_id = ?", conn.ID).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("diagnosis rows=%d want 1", count)
	}
}

func createGitHubSecret(t *testing.T, r http.Handler) string {
	t.Helper()
	body := []byte(`{"name":"GITHUB_TOKEN","value":"ghp_test","scope":{"provider":"github"}}`)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/secrets", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("secret status=%d want 201 body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	return resp.ID
}
