package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ash-repwiki/ash/internal/store"
)

func TestObserveLensesAPI(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "dev")
	r, db := newPlatformTestRouter(t)
	now := time.Now().UTC()
	run := store.RunRecord{
		ID: "run_observe_api", TraceID: "trace_observe_api", SpaceID: "local",
		ScenarioName: "feature_delivery", ScenarioVersion: "1.0.0",
		Status: "running", StartedAt: now, CreatedAt: now, UpdatedAt: now,
	}
	if err := db.Create(&run).Error; err != nil {
		t.Fatal(err)
	}
	mem := store.MemoryRecord{
		ID: "mem_observe_api", SpaceID: "local", Layer: "L1", Status: "candidate",
		Title: "t", Body: "b", TagsJSON: "[]", CreatedAt: now, UpdatedAt: now,
	}
	if err := db.Create(&mem).Error; err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/observability/lenses/global", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("global status=%d body=%s", w.Code, w.Body.String())
	}
	var global struct {
		RunningRuns int `json:"runningRuns"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &global); err != nil || global.RunningRuns < 1 {
		t.Fatalf("global=%+v err=%v body=%s", global, err, w.Body.String())
	}

	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/observability/lenses/agent?runId="+run.ID, nil)
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("agent status=%d body=%s", w2.Code, w2.Body.String())
	}

	w3 := httptest.NewRecorder()
	req3 := httptest.NewRequest(http.MethodGet, "/api/v1/observability/lenses/memory?id="+mem.ID, nil)
	r.ServeHTTP(w3, req3)
	if w3.Code != http.StatusOK {
		t.Fatalf("memory status=%d body=%s", w3.Code, w3.Body.String())
	}
}
