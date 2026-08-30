package api

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGitHubWebhookHMACAndDiagnose(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "dev")
	t.Setenv("ASH_AGENT_EXECUTOR", "static")
	r, _ := newPlatformTestRouter(t)
	secretID := createNamedSecret(t, r, "WEBHOOK_SECRET", "whsec_ds")

	connBody := []byte(`{"provider":"github","owner":"acer","repo":"ash","secretId":"` + secretID + `"}`)
	connResp := httptest.NewRecorder()
	connReq := httptest.NewRequest(http.MethodPost, "/api/v1/repo/connections", bytes.NewReader(connBody))
	connReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(connResp, connReq)
	if connResp.Code != http.StatusCreated {
		t.Fatalf("conn status=%d body=%s", connResp.Code, connResp.Body.String())
	}
	var conn struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(connResp.Body.Bytes(), &conn); err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC()
	payload, _ := json.Marshal(map[string]any{
		"action": "completed",
		"workflow_run": map[string]any{
			"id": 9001, "name": "CI", "status": "completed", "conclusion": "failure",
			"run_attempt": 1, "html_url": "https://example.test/run/9001",
			"head_branch": "main", "head_sha": "deadbeef",
			"created_at": now.Format(time.RFC3339), "updated_at": now.Format(time.RFC3339),
		},
		"repository": map[string]any{
			"name": "ash", "owner": map[string]any{"login": "acer"},
		},
	})

	bad := httptest.NewRecorder()
	badReq := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/github?connectionId="+conn.ID, bytes.NewReader(payload))
	badReq.Header.Set("Content-Type", "application/json")
	badReq.Header.Set("X-Hub-Signature-256", "sha256=00")
	badReq.Header.Set("X-GitHub-Event", "workflow_run")
	badReq.Header.Set("X-GitHub-Delivery", "deliv-bad")
	r.ServeHTTP(bad, badReq)
	if bad.Code != http.StatusUnauthorized {
		t.Fatalf("bad sig status=%d want 401 body=%s", bad.Code, bad.Body.String())
	}

	sig := signGitHubBody("whsec_ds", payload)
	ok := httptest.NewRecorder()
	okReq := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/github?connectionId="+conn.ID, bytes.NewReader(payload))
	okReq.Header.Set("Content-Type", "application/json")
	okReq.Header.Set("X-Hub-Signature-256", sig)
	okReq.Header.Set("X-GitHub-Event", "workflow_run")
	okReq.Header.Set("X-GitHub-Delivery", "deliv-1")
	r.ServeHTTP(ok, okReq)
	if ok.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", ok.Code, ok.Body.String())
	}
	var resp githubWebhookResponse
	if err := json.Unmarshal(ok.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Diagnosis == nil || resp.Diagnosis.RootCause != "test_failure" {
		t.Fatalf("resp=%+v", resp)
	}
	if resp.AshRunID != "" {
		t.Fatalf("autoRun off must not create run: %+v", resp)
	}

	dup := httptest.NewRecorder()
	dupReq := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/github?connectionId="+conn.ID, bytes.NewReader(payload))
	dupReq.Header.Set("Content-Type", "application/json")
	dupReq.Header.Set("X-Hub-Signature-256", sig)
	dupReq.Header.Set("X-GitHub-Event", "workflow_run")
	dupReq.Header.Set("X-GitHub-Delivery", "deliv-1")
	r.ServeHTTP(dup, dupReq)
	if dup.Code != http.StatusOK {
		t.Fatalf("dup status=%d body=%s", dup.Code, dup.Body.String())
	}
	var dupResp githubWebhookResponse
	if err := json.Unmarshal(dup.Body.Bytes(), &dupResp); err != nil {
		t.Fatal(err)
	}
	if !dupResp.Duplicate {
		t.Fatalf("dupResp=%+v want duplicate", dupResp)
	}
}

func TestGitHubWebhookAutoRun(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "dev")
	t.Setenv("ASH_AGENT_EXECUTOR", "static")
	r, _ := newPlatformTestRouter(t)
	secretID := createNamedSecret(t, r, "WEBHOOK_SECRET", "whsec_auto")

	connBody := []byte(`{"provider":"github","owner":"acer","repo":"ash","secretId":"` + secretID + `"}`)
	connResp := httptest.NewRecorder()
	connReq := httptest.NewRequest(http.MethodPost, "/api/v1/repo/connections", bytes.NewReader(connBody))
	connReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(connResp, connReq)
	if connResp.Code != http.StatusCreated {
		t.Fatalf("conn status=%d body=%s", connResp.Code, connResp.Body.String())
	}
	var conn struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(connResp.Body.Bytes(), &conn)

	repoRoot := t.TempDir()
	now := time.Now().UTC()
	payload, _ := json.Marshal(map[string]any{
		"action": "completed",
		"workflow_run": map[string]any{
			"id": 9002, "name": "CI", "status": "completed", "conclusion": "failure",
			"run_attempt": 1, "html_url": "https://example.test/run/9002",
			"head_branch": "main", "head_sha": "cafebabe",
			"created_at": now.Format(time.RFC3339), "updated_at": now.Format(time.RFC3339),
		},
		"repository": map[string]any{
			"name": "ash", "owner": map[string]any{"login": "acer"},
		},
	})
	sig := signGitHubBody("whsec_auto", payload)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost,
		"/api/v1/webhooks/github?connectionId="+conn.ID+"&autoRun=1&repoRoot="+repoRoot,
		bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Hub-Signature-256", sig)
	req.Header.Set("X-GitHub-Event", "workflow_run")
	req.Header.Set("X-GitHub-Delivery", "deliv-auto")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var resp githubWebhookResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if !resp.ShouldStartRun || resp.AshRunID == "" {
		t.Fatalf("resp=%+v want ashRunId", resp)
	}
}

func signGitHubBody(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func createNamedSecret(t *testing.T, r http.Handler, name, value string) string {
	t.Helper()
	body, _ := json.Marshal(map[string]any{
		"name": name, "value": value, "scope": map[string]any{"provider": "github"},
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/secrets", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("secret status=%d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	return resp.ID
}
