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
		ID             string `json:"id"`
		RootCause      string `json:"rootCause"`
		DecisionStatus string `json:"decisionStatus"`
	}
	if err := json.Unmarshal(diagResp.Body.Bytes(), &diag); err != nil {
		t.Fatal(err)
	}
	if diag.RootCause != "test_failure" {
		t.Fatalf("diag=%+v want test_failure", diag)
	}
	listResp := httptest.NewRecorder()
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/ci/diagnoses?connectionId="+conn.ID, nil)
	r.ServeHTTP(listResp, listReq)
	if listResp.Code != http.StatusOK {
		t.Fatalf("list status=%d want 200 body=%s", listResp.Code, listResp.Body.String())
	}
	var list struct {
		Items []struct {
			ID             string `json:"id"`
			DecisionStatus string `json:"decisionStatus"`
		} `json:"items"`
	}
	if err := json.Unmarshal(listResp.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if len(list.Items) != 1 || list.Items[0].DecisionStatus != "pending" {
		t.Fatalf("list=%+v want pending diagnosis", list)
	}
	adoptResp := httptest.NewRecorder()
	adoptReq := httptest.NewRequest(http.MethodPost, "/api/v1/ci/diagnoses/"+diag.ID+"/adopt", bytes.NewReader([]byte(`{"reason":"fix matched"}`)))
	adoptReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(adoptResp, adoptReq)
	if adoptResp.Code != http.StatusOK {
		t.Fatalf("adopt status=%d want 200 body=%s", adoptResp.Code, adoptResp.Body.String())
	}
	var adopted struct {
		Adopted        bool   `json:"adopted"`
		DecisionStatus string `json:"decisionStatus"`
	}
	if err := json.Unmarshal(adoptResp.Body.Bytes(), &adopted); err != nil {
		t.Fatal(err)
	}
	if !adopted.Adopted || adopted.DecisionStatus != "adopted" {
		t.Fatalf("adopted=%+v want adopted", adopted)
	}
	var count int64
	if err := db.Model(&store.CIDiagnosis{}).Where("connection_id = ?", conn.ID).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("diagnosis rows=%d want 1", count)
	}
}

func TestCISyncRunsWithFixture(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "dev")
	t.Setenv("ASH_CI_FIXTURE", "1")
	r, _ := newPlatformTestRouter(t)
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

	runsResp := httptest.NewRecorder()
	runsReq := httptest.NewRequest(http.MethodGet, "/api/v1/ci/runs?connectionId="+conn.ID+"&sync=true", nil)
	r.ServeHTTP(runsResp, runsReq)
	if runsResp.Code != http.StatusOK {
		t.Fatalf("runs status=%d want 200 body=%s", runsResp.Code, runsResp.Body.String())
	}
	var runs struct {
		Items []store.CIRun `json:"items"`
	}
	if err := json.Unmarshal(runsResp.Body.Bytes(), &runs); err != nil {
		t.Fatal(err)
	}
	if len(runs.Items) != 1 || runs.Items[0].ProviderRunID != "fixture-run-9001" {
		t.Fatalf("runs=%+v want fixture-run-9001", runs.Items)
	}

	jobsResp := httptest.NewRecorder()
	jobsReq := httptest.NewRequest(http.MethodGet, "/api/v1/ci/jobs?runId="+runs.Items[0].ID+"&sync=true", nil)
	r.ServeHTTP(jobsResp, jobsReq)
	if jobsResp.Code != http.StatusOK {
		t.Fatalf("jobs status=%d want 200 body=%s", jobsResp.Code, jobsResp.Body.String())
	}
	var jobs struct {
		Items []store.CIJob `json:"items"`
	}
	if err := json.Unmarshal(jobsResp.Body.Bytes(), &jobs); err != nil {
		t.Fatal(err)
	}
	if len(jobs.Items) != 2 || jobs.Items[0].ProviderJobID != "fixture-job-9101" {
		t.Fatalf("jobs=%+v want 2 fixture jobs", jobs.Items)
	}

	diagBody := []byte(`{"connectionId":"` + conn.ID + `","runId":"` + runs.Items[0].ID + `","jobId":"` + jobs.Items[0].ID + `"}`)
	diagResp := httptest.NewRecorder()
	diagReq := httptest.NewRequest(http.MethodPost, "/api/v1/ci/failures/diagnose", bytes.NewReader(diagBody))
	diagReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(diagResp, diagReq)
	if diagResp.Code != http.StatusOK {
		t.Fatalf("diagnose status=%d want 200 body=%s", diagResp.Code, diagResp.Body.String())
	}
	var diag struct {
		ID             string `json:"id"`
		RootCause      string `json:"rootCause"`
		LogDigest      string `json:"logDigest"`
		DecisionStatus string `json:"decisionStatus"`
	}
	if err := json.Unmarshal(diagResp.Body.Bytes(), &diag); err != nil {
		t.Fatal(err)
	}
	if diag.RootCause != "test_failure" || diag.LogDigest == "" || diag.DecisionStatus != "pending" {
		t.Fatalf("diag=%+v want pending test_failure with digest", diag)
	}

	listResp := httptest.NewRecorder()
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/ci/diagnoses?connectionId="+conn.ID+"&jobId="+jobs.Items[0].ID, nil)
	r.ServeHTTP(listResp, listReq)
	if listResp.Code != http.StatusOK {
		t.Fatalf("list status=%d want 200 body=%s", listResp.Code, listResp.Body.String())
	}
	var list struct {
		Items []struct {
			ID        string `json:"id"`
			LogDigest string `json:"logDigest"`
		} `json:"items"`
	}
	if err := json.Unmarshal(listResp.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if len(list.Items) != 1 || list.Items[0].ID != diag.ID || list.Items[0].LogDigest == "" {
		t.Fatalf("list=%+v want diagnosis with digest", list)
	}

	adoptResp := httptest.NewRecorder()
	adoptReq := httptest.NewRequest(http.MethodPost, "/api/v1/ci/diagnoses/"+diag.ID+"/adopt", bytes.NewReader([]byte(`{"reason":"fixture matched"}`)))
	adoptReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(adoptResp, adoptReq)
	if adoptResp.Code != http.StatusOK {
		t.Fatalf("adopt status=%d want 200 body=%s", adoptResp.Code, adoptResp.Body.String())
	}

	jobsResp2 := httptest.NewRecorder()
	jobsReq2 := httptest.NewRequest(http.MethodGet, "/api/v1/ci/jobs?runId="+runs.Items[0].ID, nil)
	r.ServeHTTP(jobsResp2, jobsReq2)
	if jobsResp2.Code != http.StatusOK {
		t.Fatalf("jobs reload status=%d body=%s", jobsResp2.Code, jobsResp2.Body.String())
	}
	var jobsReload struct {
		Items []store.CIJob `json:"items"`
	}
	if err := json.Unmarshal(jobsResp2.Body.Bytes(), &jobsReload); err != nil {
		t.Fatal(err)
	}
	if len(jobsReload.Items) < 1 || jobsReload.Items[0].LogDigest != diag.LogDigest {
		t.Fatalf("jobs=%+v want logDigest=%q on synced job", jobsReload.Items, diag.LogDigest)
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
