package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ash-repwiki/ash/internal/store"
)

func TestGetSpaceQuotas(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "dev")
	r, db := newPlatformTestRouter(t)
	now := time.Now().UTC()
	if err := db.Create(&store.SpacePolicyPack{
		SpaceID: "local", CitationMode: "optional",
		BodyJSON:  `{"quotas":{"maxConcurrentRuns":2,"tokenBudgetProxy":99}}`,
		CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&store.RunRecord{
		ID: "run_q_active", TraceID: "trc_q", ScenarioName: "feature_delivery", ScenarioVersion: "1.0.0",
		PolicyProfile: "default", Status: "running", SpaceID: "local",
		StartedAt: now, CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/spaces/local/quotas", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		SpaceID string `json:"spaceId"`
		Limits  struct {
			MaxConcurrentRuns int `json:"maxConcurrentRuns"`
			TokenBudgetProxy  int `json:"tokenBudgetProxy"`
		} `json:"limits"`
		Usage struct {
			ActiveConcurrentRuns int `json:"activeConcurrentRuns"`
			TokenBudgetProxyUsed int `json:"tokenBudgetProxyUsed"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.SpaceID != "local" || resp.Limits.MaxConcurrentRuns != 2 || resp.Limits.TokenBudgetProxy != 99 {
		t.Fatalf("%+v", resp)
	}
	if resp.Usage.ActiveConcurrentRuns != 1 || resp.Usage.TokenBudgetProxyUsed != 0 {
		t.Fatalf("usage=%+v", resp.Usage)
	}
}
