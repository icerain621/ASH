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

func TestObservabilityAlertsAndMetricsAPI(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "dev")
	r, db := newPlatformTestRouter(t)
	now := time.Now().UTC()
	if err := db.Create(&store.Feedback{
		ID: "fb_api_low", SpaceID: "local", TargetType: "run", TargetID: "run_low",
		Rating: 1, Category: "quality", Status: "open", Severity: "warn", Source: "test",
		CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	evalResp := httptest.NewRecorder()
	evalReq := httptest.NewRequest(http.MethodPost, "/api/v1/observability/alerts/evaluate", nil)
	r.ServeHTTP(evalResp, evalReq)
	if evalResp.Code != http.StatusOK {
		t.Fatalf("eval status=%d want 200 body=%s", evalResp.Code, evalResp.Body.String())
	}
	if !bytes.Contains(evalResp.Body.Bytes(), []byte("low_feedback_rate")) {
		t.Fatalf("eval body=%s want low_feedback_rate", evalResp.Body.String())
	}

	metricsResp := httptest.NewRecorder()
	metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	r.ServeHTTP(metricsResp, metricsReq)
	if metricsResp.Code != http.StatusOK {
		t.Fatalf("metrics status=%d want 200 body=%s", metricsResp.Code, metricsResp.Body.String())
	}
	if metricsResp.Header().Get("X-Ash-Metrics-Scope") != "global" {
		t.Fatalf("metrics scope header=%q want global", metricsResp.Header().Get("X-Ash-Metrics-Scope"))
	}
	if !strings.Contains(metricsResp.Body.String(), "feedback_low_score_total 1") {
		t.Fatalf("metrics=%s want feedback_low_score_total", metricsResp.Body.String())
	}

	scopedResp := httptest.NewRecorder()
	scopedReq := httptest.NewRequest(http.MethodGet, "/api/v1/metrics/prometheus?spaceId=local", nil)
	r.ServeHTTP(scopedResp, scopedReq)
	if scopedResp.Code != http.StatusOK {
		t.Fatalf("scoped metrics status=%d want 200 body=%s", scopedResp.Code, scopedResp.Body.String())
	}
	if scopedResp.Header().Get("X-Ash-Metrics-Scope") != "space:local" {
		t.Fatalf("scoped scope header=%q want space:local", scopedResp.Header().Get("X-Ash-Metrics-Scope"))
	}
	if !strings.Contains(scopedResp.Body.String(), `space_id="local"`) {
		t.Fatalf("scoped metrics=%s want space_id label", scopedResp.Body.String())
	}
}

func TestReleaseGovernanceAPI(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "dev")
	r, db := newPlatformTestRouter(t)
	now := time.Now().UTC()
	seedReleaseGateEvidence(t, db, now)

	createResp := httptest.NewRecorder()
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/releases", bytes.NewReader([]byte(`{"version":"v0.4.0","title":"Back four closure"}`)))
	createReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(createResp, createReq)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("create status=%d want 201 body=%s", createResp.Code, createResp.Body.String())
	}
	var rel store.ReleaseRecord
	if err := json.Unmarshal(createResp.Body.Bytes(), &rel); err != nil {
		t.Fatal(err)
	}
	checkResp := httptest.NewRecorder()
	checkReq := httptest.NewRequest(http.MethodGet, "/api/v1/releases/"+rel.ID+"/checklist", nil)
	r.ServeHTTP(checkResp, checkReq)
	if checkResp.Code != http.StatusOK {
		t.Fatalf("checklist status=%d want 200 body=%s", checkResp.Code, checkResp.Body.String())
	}
	var checklist struct {
		Items []store.ReleaseChecklistItem `json:"items"`
	}
	if err := json.Unmarshal(checkResp.Body.Bytes(), &checklist); err != nil {
		t.Fatal(err)
	}
	if len(checklist.Items) == 0 {
		t.Fatal("expected checklist items")
	}

	gateResp := httptest.NewRecorder()
	gateReq := httptest.NewRequest(http.MethodPost, "/api/v1/releases/"+rel.ID+"/gate", nil)
	r.ServeHTTP(gateResp, gateReq)
	if gateResp.Code != http.StatusOK {
		t.Fatalf("gate status=%d want 200 body=%s", gateResp.Code, gateResp.Body.String())
	}
	if !bytes.Contains(gateResp.Body.Bytes(), []byte(`"overall":"pass"`)) {
		t.Fatalf("gate body=%s want pass", gateResp.Body.String())
	}

	drillResp := httptest.NewRecorder()
	drillReq := httptest.NewRequest(http.MethodPost, "/api/v1/releases/"+rel.ID+"/rollback-drills", bytes.NewReader([]byte(`{"scenario":"rollback image","status":"passed","durationMs":60000}`)))
	drillReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(drillResp, drillReq)
	if drillResp.Code != http.StatusCreated {
		t.Fatalf("drill status=%d want 201 body=%s", drillResp.Code, drillResp.Body.String())
	}
}

func seedReleaseGateEvidence(t *testing.T, db *store.DB, now time.Time) {
	t.Helper()
	done := now.Add(time.Minute)
	rows := []any{
		&store.CIRun{
			ID: "ci_api_release_pass", SpaceID: "local", ConnectionID: "conn_release", ProviderRunID: "43",
			Workflow: "ci", Status: "completed", Conclusion: "success", Attempt: 1,
			StartedAt: &now, CompletedAt: &done, CreatedAt: now, UpdatedAt: done,
		},
		&store.AuditLog{ID: "aud_api_doctor_m3", SpaceID: "local", EventType: "doctor.suite_completed", PayloadJSON: `{"suite":"M3","pass":10,"fail":0}`, CreatedAt: now},
		&store.AuditLog{ID: "aud_api_doctor_all", SpaceID: "local", EventType: "doctor.suite_completed", PayloadJSON: `{"suite":"ALL","pass":30,"fail":0}`, CreatedAt: now},
		&store.AuditLog{ID: "aud_api_pg", SpaceID: "local", EventType: "postgres.e2e_completed", PayloadJSON: `{"status":"pass"}`, CreatedAt: now},
		&store.AuditLog{ID: "aud_api_execgo", SpaceID: "local", EventType: "execgo.live_smoke", PayloadJSON: `{"status":"pass"}`, CreatedAt: now},
		&store.Feedback{ID: "fb_api_release_ok", SpaceID: "local", TargetType: "release", TargetID: "rel", Rating: 5, Category: "quality", Status: "resolved", Severity: "info", Source: "test", CreatedAt: now, UpdatedAt: now},
	}
	for _, row := range rows {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
}
