package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ash-repwiki/ash/internal/events"
	"github.com/ash-repwiki/ash/internal/memory"
	"github.com/ash-repwiki/ash/internal/secrets"
	"github.com/ash-repwiki/ash/internal/store"
)

// TestReleaseSamplingH09 exercises postgres-rds-e2e.md §7 API paths without a live worker.
func TestReleaseSamplingH09(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "dev")
	r, db := newPlatformTestRouter(t)
	now := time.Now().UTC()
	space := "space_h09"

	run := store.RunRecord{
		ID: "run_h09", TraceID: "trace_h09", ScenarioName: "feature_delivery", ScenarioVersion: "1.0.0",
		PolicyProfile: "default", Status: "completed", SpaceID: space,
		StartedAt: now, CreatedAt: now, UpdatedAt: now,
	}
	if err := db.Create(&run).Error; err != nil {
		t.Fatal(err)
	}
	getRun := httptest.NewRecorder()
	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/runs/"+run.ID, nil)
	getReq.Header.Set("X-ASH-Space-ID", space)
	r.ServeHTTP(getRun, getReq)
	if getRun.Code != http.StatusOK {
		t.Fatalf("7.1 run status=%d body=%s", getRun.Code, getRun.Body.String())
	}

	candBody := []byte(`{"layer":"L1","title":"H09 probe","body":"release sampling memory","scopeRepo":"ash","evidence":[{"kind":"file","ref":"doc/h09.md"}]}`)
	candResp := httptest.NewRecorder()
	candReq := httptest.NewRequest(http.MethodPost, "/api/v1/memory/candidates", bytes.NewReader(candBody))
	candReq.Header.Set("Content-Type", "application/json")
	candReq.Header.Set("X-ASH-Space-ID", space)
	r.ServeHTTP(candResp, candReq)
	if candResp.Code != http.StatusCreated {
		t.Fatalf("7.3 candidate status=%d body=%s", candResp.Code, candResp.Body.String())
	}
	var cand struct {
		CandidateID string `json:"candidateId"`
	}
	if err := json.Unmarshal(candResp.Body.Bytes(), &cand); err != nil {
		t.Fatal(err)
	}
	reviewBody := []byte(`{"decision":"approve","reason":"h09","policyProfile":"default"}`)
	reviewResp := httptest.NewRecorder()
	reviewReq := httptest.NewRequest(http.MethodPost, "/api/v1/memory/candidates/"+cand.CandidateID+"/review", bytes.NewReader(reviewBody))
	reviewReq.Header.Set("Content-Type", "application/json")
	reviewReq.Header.Set("X-ASH-Space-ID", space)
	r.ServeHTTP(reviewResp, reviewReq)
	if reviewResp.Code != http.StatusOK {
		t.Fatalf("7.3 review status=%d body=%s", reviewResp.Code, reviewResp.Body.String())
	}
	queryBody := []byte(`{"text":"H09 probe","topK":5}`)
	queryResp := httptest.NewRecorder()
	queryReq := httptest.NewRequest(http.MethodPost, "/api/v1/memory/query", bytes.NewReader(queryBody))
	queryReq.Header.Set("Content-Type", "application/json")
	queryReq.Header.Set("X-ASH-Space-ID", space)
	r.ServeHTTP(queryResp, queryReq)
	if queryResp.Code != http.StatusOK {
		t.Fatalf("7.3 query status=%d body=%s", queryResp.Code, queryResp.Body.String())
	}

	ttlQueueResp := httptest.NewRecorder()
	ttlQueueReq := httptest.NewRequest(http.MethodGet, "/api/v1/memory/ttl-queue?limit=10", nil)
	ttlQueueReq.Header.Set("X-ASH-Space-ID", space)
	r.ServeHTTP(ttlQueueResp, ttlQueueReq)
	if ttlQueueResp.Code != http.StatusOK {
		t.Fatalf("7.3 ttl-queue status=%d body=%s", ttlQueueResp.Code, ttlQueueResp.Body.String())
	}

	kpiResp := httptest.NewRecorder()
	kpiReq := httptest.NewRequest(http.MethodGet, "/api/v1/metrics/overview?spaceId="+space, nil)
	kpiReq.Header.Set("X-ASH-Space-ID", space)
	r.ServeHTTP(kpiResp, kpiReq)
	if kpiResp.Code != http.StatusOK {
		t.Fatalf("7.4 metrics status=%d body=%s", kpiResp.Code, kpiResp.Body.String())
	}

	diagBody := []byte(`{"logText":"go test ./...\n--- FAIL: TestH09 (0.01s)\nFAIL\tpkg\t0.1s"}`)
	diagResp := httptest.NewRecorder()
	diagReq := httptest.NewRequest(http.MethodPost, "/api/v1/ci/failures/diagnose", bytes.NewReader(diagBody))
	diagReq.Header.Set("Content-Type", "application/json")
	diagReq.Header.Set("X-ASH-Space-ID", space)
	r.ServeHTTP(diagResp, diagReq)
	if diagResp.Code != http.StatusOK {
		t.Fatalf("7.5 diagnose status=%d body=%s", diagResp.Code, diagResp.Body.String())
	}
	var diagCount int64
	if err := db.Model(&store.CIDiagnosis{}).Where("space_id = ?", space).Count(&diagCount).Error; err != nil {
		t.Fatal(err)
	}
	if diagCount != 1 {
		t.Fatalf("7.5 diagnoses=%d want 1", diagCount)
	}

	_ = db.Create(&store.AuditPolicy{SpaceID: space, RetentionDays: 30, CreatedAt: now, UpdatedAt: now}).Error
	exportBody := []byte(`{"suite":"TR2"}`)
	exportResp := httptest.NewRecorder()
	exportReq := httptest.NewRequest(http.MethodPost, "/api/v1/compliance/export", bytes.NewReader(exportBody))
	exportReq.Header.Set("Content-Type", "application/json")
	exportReq.Header.Set("X-ASH-Space-ID", space)
	r.ServeHTTP(exportResp, exportReq)
	if exportResp.Code != http.StatusAccepted {
		t.Fatalf("7.6 export status=%d body=%s", exportResp.Code, exportResp.Body.String())
	}

	scaleResp := httptest.NewRecorder()
	scaleReq := httptest.NewRequest(http.MethodGet, "/api/v1/scale/readiness", nil)
	scaleReq.Header.Set("X-ASH-Space-ID", space)
	r.ServeHTTP(scaleResp, scaleReq)
	if scaleResp.Code != http.StatusOK {
		t.Fatalf("7.7 scale status=%d body=%s", scaleResp.Code, scaleResp.Body.String())
	}
	var scale ScaleReadinessResponse
	if err := json.Unmarshal(scaleResp.Body.Bytes(), &scale); err != nil {
		t.Fatal(err)
	}
	if scale.MemorySchemaVersion != memory.CurrentSchemaVersion {
		t.Fatalf("memorySchemaVersion=%d want %d", scale.MemorySchemaVersion, memory.CurrentSchemaVersion)
	}

	migResp, err := memory.NewService(db, nil).RunMigrations(memory.RunMigrationRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if !migResp.OK {
		t.Fatalf("memory migrate=%+v", migResp)
	}
}

// TestReleaseSamplingSSE covers postgres-rds-e2e.md §7.2 (stream + audit).
func TestReleaseSamplingSSE(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "dev")
	r, db := newPlatformTestRouter(t)
	now := time.Now().UTC()
	space := "space_sse_h09"
	runID := "run_sse_h09"
	traceID := "trace_sse_h09"

	if err := db.Create(&store.RunRecord{
		ID: runID, TraceID: traceID, ScenarioName: "feature_delivery", ScenarioVersion: "1.0.0",
		PolicyProfile: "default", Status: "completed", SpaceID: space,
		StartedAt: now, CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	ev := events.NewService(db)
	if _, err := ev.Append(runID, traceID, "run.started", "info", map[string]any{"probe": "h09"}); err != nil {
		t.Fatal(err)
	}
	if _, err := ev.Append(runID, traceID, "step.completed", "info", map[string]any{"step": "arch.design"}); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/runs/"+runID+"/stream", nil)
	req = req.WithContext(ctx)
	req.Header.Set("X-ASH-Space-ID", space)

	done := make(chan struct{})
	w := httptest.NewRecorder()
	go func() {
		r.ServeHTTP(w, req)
		close(done)
	}()
	time.Sleep(300 * time.Millisecond)
	cancel()
	<-done

	if ct := w.Header().Get("Content-Type"); !strings.Contains(ct, "text/event-stream") {
		t.Fatalf("content-type=%q want text/event-stream", ct)
	}
	body := w.Body.String()
	if !strings.Contains(body, "data:") || !strings.Contains(body, "run.started") {
		t.Fatalf("body=%q want SSE data with run.started", body)
	}

	var opened int64
	if err := db.Model(&store.AuditLog{}).
		Where("space_id = ? AND event_type = ?", space, "stream.session_opened").
		Count(&opened).Error; err != nil {
		t.Fatal(err)
	}
	if opened != 1 {
		t.Fatalf("stream.session_opened audits=%d want 1", opened)
	}
}

// TestStreamRunResumesFromQueryLastEventID verifies Sprint BM query resume (R-07).
func TestStreamRunResumesFromQueryLastEventID(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "dev")
	r, db := newPlatformTestRouter(t)
	now := time.Now().UTC()
	space := "space_sse_resume"
	runID := "run_sse_resume"
	traceID := "trace_sse_resume"

	if err := db.Create(&store.RunRecord{
		ID: runID, TraceID: traceID, ScenarioName: "feature_delivery", ScenarioVersion: "1.0.0",
		PolicyProfile: "default", Status: "running", SpaceID: space,
		StartedAt: now, CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	ev := events.NewService(db)
	first, err := ev.Append(runID, traceID, "run.started", "info", map[string]any{"n": 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ev.Append(runID, traceID, "step.started", "info", map[string]any{"n": 2}); err != nil {
		t.Fatal(err)
	}
	if _, err := ev.Append(runID, traceID, "step.finished", "info", map[string]any{"n": 3}); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/runs/"+runID+"/stream?Last-Event-ID="+first.ID, nil)
	req = req.WithContext(ctx)
	req.Header.Set("X-ASH-Space-ID", space)

	done := make(chan struct{})
	w := httptest.NewRecorder()
	go func() {
		r.ServeHTTP(w, req)
		close(done)
	}()
	time.Sleep(300 * time.Millisecond)
	cancel()
	<-done

	body := w.Body.String()
	if !strings.Contains(body, "step.started") || !strings.Contains(body, "step.finished") {
		t.Fatalf("body=%q want events after Last-Event-ID", body)
	}
	if strings.Contains(body, first.ID) || strings.Contains(body, "run.started") {
		t.Fatalf("body=%q should skip resumed-from event %s", body, first.ID)
	}
}

// TestReleaseSamplingH09CrossSpaceMemoryDenied is §7.3 cross-space isolation adjunct.
func TestReleaseSamplingH09CrossSpaceMemoryDenied(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "dev")
	r, db := newPlatformTestRouter(t)
	now := time.Now().UTC()
	other := "space_h09_other"
	if err := db.Create(&store.MemoryRecord{
		ID: "mem_h09_other", Layer: "L0", Status: "approved", SpaceID: other,
		SchemaVersion: 2, Title: "secret", Body: "other tenant", TagsJSON: "[]",
		Sensitivity: "normal", CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	body := []byte(`{"text":"secret","topK":5}`)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/memory/query", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-ASH-Space-ID", "space_h09")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	for _, item := range resp.Items {
		if item.ID == "mem_h09_other" {
			t.Fatalf("cross-space memory leaked: %+v", resp.Items)
		}
	}
}

// TestReleaseSamplingCIFixtureH04H05 exercises H-04/H-05 via ASH_CI_FIXTURE without GitHub API.
func TestReleaseSamplingCIFixtureH04H05(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "dev")
	t.Setenv("ASH_CI_FIXTURE", "1")
	r, _ := newPlatformTestRouter(t)
	space := "space_ci_fixture"

	secretBody := []byte(`{"name":"GITHUB_TOKEN","value":"ghp_fixture","scope":{"provider":"github"}}`)
	secretResp := httptest.NewRecorder()
	secretReq := httptest.NewRequest(http.MethodPost, "/api/v1/secrets", bytes.NewReader(secretBody))
	secretReq.Header.Set("Content-Type", "application/json")
	secretReq.Header.Set("X-ASH-Space-ID", space)
	r.ServeHTTP(secretResp, secretReq)
	if secretResp.Code != http.StatusCreated {
		t.Fatalf("secret status=%d body=%s", secretResp.Code, secretResp.Body.String())
	}
	var secret struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(secretResp.Body.Bytes(), &secret); err != nil {
		t.Fatal(err)
	}

	connBody := []byte(`{"provider":"github","owner":"iammm0","repo":"ASH","secretId":"` + secret.ID + `"}`)
	connResp := httptest.NewRecorder()
	connReq := httptest.NewRequest(http.MethodPost, "/api/v1/repo/connections", bytes.NewReader(connBody))
	connReq.Header.Set("Content-Type", "application/json")
	connReq.Header.Set("X-ASH-Space-ID", space)
	r.ServeHTTP(connResp, connReq)
	if connResp.Code != http.StatusCreated {
		t.Fatalf("connection status=%d body=%s", connResp.Code, connResp.Body.String())
	}
	var conn struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(connResp.Body.Bytes(), &conn); err != nil {
		t.Fatal(err)
	}

	runsResp := httptest.NewRecorder()
	runsReq := httptest.NewRequest(http.MethodGet, "/api/v1/ci/runs?connectionId="+conn.ID+"&sync=true", nil)
	runsReq.Header.Set("X-ASH-Space-ID", space)
	r.ServeHTTP(runsResp, runsReq)
	if runsResp.Code != http.StatusOK {
		t.Fatalf("H-04 runs status=%d body=%s", runsResp.Code, runsResp.Body.String())
	}
	var runs struct {
		Items []struct {
			ID            string `json:"id"`
			ProviderRunID string `json:"providerRunId"`
		} `json:"items"`
	}
	if err := json.Unmarshal(runsResp.Body.Bytes(), &runs); err != nil {
		t.Fatal(err)
	}
	if len(runs.Items) != 1 || runs.Items[0].ProviderRunID != "fixture-run-9001" {
		t.Fatalf("runs=%+v want fixture-run-9001", runs.Items)
	}

	jobsResp := httptest.NewRecorder()
	jobsReq := httptest.NewRequest(http.MethodGet, "/api/v1/ci/jobs?runId="+runs.Items[0].ID+"&sync=true", nil)
	jobsReq.Header.Set("X-ASH-Space-ID", space)
	r.ServeHTTP(jobsResp, jobsReq)
	if jobsResp.Code != http.StatusOK {
		t.Fatalf("H-05 jobs status=%d body=%s", jobsResp.Code, jobsResp.Body.String())
	}
	var jobs struct {
		Items []struct {
			ID            string `json:"id"`
			ProviderJobID string `json:"providerJobId"`
		} `json:"items"`
	}
	if err := json.Unmarshal(jobsResp.Body.Bytes(), &jobs); err != nil {
		t.Fatal(err)
	}
	if len(jobs.Items) != 2 {
		t.Fatalf("jobs=%+v want 2 fixture jobs", jobs.Items)
	}

	diagBody := []byte(`{"jobId":"` + jobs.Items[1].ID + `"}`)
	diagResp := httptest.NewRecorder()
	diagReq := httptest.NewRequest(http.MethodPost, "/api/v1/ci/failures/diagnose", bytes.NewReader(diagBody))
	diagReq.Header.Set("Content-Type", "application/json")
	diagReq.Header.Set("X-ASH-Space-ID", space)
	r.ServeHTTP(diagResp, diagReq)
	if diagResp.Code != http.StatusOK {
		t.Fatalf("H-05 diagnose status=%d body=%s", diagResp.Code, diagResp.Body.String())
	}
	var diag struct {
		RootCause string `json:"rootCause"`
		LogDigest string `json:"logDigest"`
	}
	if err := json.Unmarshal(diagResp.Body.Bytes(), &diag); err != nil {
		t.Fatal(err)
	}
	if diag.RootCause != "docker_or_postgres_unavailable" || diag.LogDigest == "" {
		t.Fatalf("diag=%+v want docker failure with digest", diag)
	}
}

// TestSecretRotateRepoConnectionH07 verifies repo connections keep working after secret rotate (H-07).
func TestSecretRotateRepoConnectionH07(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "dev")
	t.Setenv("ASH_CI_FIXTURE", "1")
	t.Setenv("ASH_SECRET_KEY", "test-secret-key-h07")
	r, db := newPlatformTestRouter(t)
	space := "space_h07_rotate"

	secretBody := []byte(`{"name":"GITHUB_TOKEN","value":"ghp_before_rotate","scope":{"provider":"github"}}`)
	secretResp := httptest.NewRecorder()
	secretReq := httptest.NewRequest(http.MethodPost, "/api/v1/secrets", bytes.NewReader(secretBody))
	secretReq.Header.Set("Content-Type", "application/json")
	secretReq.Header.Set("X-ASH-Space-ID", space)
	r.ServeHTTP(secretResp, secretReq)
	if secretResp.Code != http.StatusCreated {
		t.Fatalf("secret status=%d body=%s", secretResp.Code, secretResp.Body.String())
	}
	var secret struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(secretResp.Body.Bytes(), &secret); err != nil {
		t.Fatal(err)
	}

	connBody := []byte(`{"provider":"github","owner":"iammm0","repo":"ASH","secretId":"` + secret.ID + `"}`)
	connResp := httptest.NewRecorder()
	connReq := httptest.NewRequest(http.MethodPost, "/api/v1/repo/connections", bytes.NewReader(connBody))
	connReq.Header.Set("Content-Type", "application/json")
	connReq.Header.Set("X-ASH-Space-ID", space)
	r.ServeHTTP(connResp, connReq)
	if connResp.Code != http.StatusCreated {
		t.Fatalf("connection status=%d body=%s", connResp.Code, connResp.Body.String())
	}
	var conn struct {
		ID       string `json:"id"`
		SecretID string `json:"secretId"`
	}
	if err := json.Unmarshal(connResp.Body.Bytes(), &conn); err != nil {
		t.Fatal(err)
	}
	if conn.SecretID != secret.ID {
		t.Fatalf("conn=%+v want secretId=%s", conn, secret.ID)
	}

	syncRuns := func() string {
		t.Helper()
		runsResp := httptest.NewRecorder()
		runsReq := httptest.NewRequest(http.MethodGet, "/api/v1/ci/runs?connectionId="+conn.ID+"&sync=true", nil)
		runsReq.Header.Set("X-ASH-Space-ID", space)
		r.ServeHTTP(runsResp, runsReq)
		if runsResp.Code != http.StatusOK {
			t.Fatalf("runs sync status=%d body=%s", runsResp.Code, runsResp.Body.String())
		}
		var runs struct {
			Items []struct {
				ID            string `json:"id"`
				ProviderRunID string `json:"providerRunId"`
			} `json:"items"`
		}
		if err := json.Unmarshal(runsResp.Body.Bytes(), &runs); err != nil {
			t.Fatal(err)
		}
		if len(runs.Items) != 1 || runs.Items[0].ProviderRunID != "fixture-run-9001" {
			t.Fatalf("runs=%+v want fixture-run-9001", runs.Items)
		}
		return runs.Items[0].ID
	}
	runID := syncRuns()

	rotateResp := httptest.NewRecorder()
	rotateReq := httptest.NewRequest(http.MethodPost, "/api/v1/secrets/"+secret.ID+"/rotate", bytes.NewReader([]byte(`{"value":"ghp_after_rotate"}`)))
	rotateReq.Header.Set("Content-Type", "application/json")
	rotateReq.Header.Set("X-ASH-Space-ID", space)
	r.ServeHTTP(rotateResp, rotateReq)
	if rotateResp.Code != http.StatusOK {
		t.Fatalf("rotate status=%d body=%s", rotateResp.Code, rotateResp.Body.String())
	}

	var row store.SecretRecord
	if err := db.First(&row, "id = ?", secret.ID).Error; err != nil {
		t.Fatal(err)
	}
	if row.ValueDigest != secrets.Digest("ghp_after_rotate") {
		t.Fatalf("digest=%q want rotated digest", row.ValueDigest)
	}

	if got := syncRuns(); got != runID {
		t.Fatalf("post-rotate run id=%s want same fixture run %s", got, runID)
	}

	var audits []store.AuditLog
	if err := db.Where("space_id = ? AND event_type = ?", space, "secret.rotated").Find(&audits).Error; err != nil {
		t.Fatal(err)
	}
	if len(audits) != 1 {
		t.Fatalf("audits=%+v want one secret.rotated", audits)
	}
	if strings.Contains(audits[0].PayloadJSON, "ghp_after_rotate") || strings.Contains(audits[0].PayloadJSON, "ghp_before_rotate") {
		t.Fatalf("audit leaked secret value: %+v", audits[0])
	}
}
