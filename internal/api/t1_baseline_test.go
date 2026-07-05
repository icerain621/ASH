package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ash-repwiki/ash/internal/metrics"
	"github.com/ash-repwiki/ash/internal/store"
)

func TestT1KPIOverviewBaseline(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "dev")
	r, db := newPlatformTestRouter(t)
	now := time.Now().UTC()
	if err := db.Create(&store.RunRecord{
		ID: "run_t1_kpi", TraceID: "trace_t1", SpaceID: "local",
		ScenarioName: "feature_delivery", ScenarioVersion: "1.0.0",
		Status: "finished", StartedAt: now, CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/metrics/overview?spaceId=local", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("overview status=%d body=%s", w.Code, w.Body.String())
	}
	var overview metrics.Overview
	if err := json.Unmarshal(w.Body.Bytes(), &overview); err != nil {
		t.Fatal(err)
	}
	if len(overview.Summary) == 0 {
		t.Fatal("expected KPI summary cards")
	}
	found := false
	for _, card := range overview.Summary {
		if card.ID == "KPI-01" {
			found = true
			if card.Value < 0 {
				t.Fatalf("KPI-01 value=%v want >=0", card.Value)
			}
		}
	}
	if !found {
		t.Fatalf("summary=%+v want KPI-01", overview.Summary)
	}
}

func TestT1FeedbackIngestBaseline(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "dev")
	r, _ := newPlatformTestRouter(t)
	body := []byte(`{"targetType":"run","targetId":"run_t1","rating":4,"category":"quality","comment":"t1 baseline"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/feedback", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("feedback status=%d body=%s", w.Code, w.Body.String())
	}
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/feedback?spaceId=local", nil)
	listResp := httptest.NewRecorder()
	r.ServeHTTP(listResp, listReq)
	if listResp.Code != http.StatusOK {
		t.Fatalf("feedback list status=%d body=%s", listResp.Code, listResp.Body.String())
	}
	if !bytes.Contains(listResp.Body.Bytes(), []byte("run_t1")) {
		t.Fatalf("feedback list=%s want run_t1", listResp.Body.Bytes())
	}
}
