package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ash-repwiki/ash/internal/rules"
	"github.com/ash-repwiki/ash/internal/runs"
	"github.com/ash-repwiki/ash/internal/store"
	"github.com/ash-repwiki/ash/internal/waker"
)

func TestWakerQueueAndSweep(t *testing.T) {
	t.Setenv("ASH_WAKER_RUN_TTL", "30m")
	gin.SetMode(gin.TestMode)
	dir := t.TempDir()
	db, err := store.Open(filepath.Join(dir, "ash.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	now := time.Now().UTC()
	if err := db.Create(&store.RunRecord{
		ID: "run_waker_api", TraceID: "tr_waker",
		ScenarioName: "feature_delivery", ScenarioVersion: "1.0.0",
		PolicyProfile: "default", Status: runs.StatusRunning, SpaceID: "local",
		RepoRoot: ".", StartedAt: now.Add(-2 * time.Hour),
		CreatedAt: now.Add(-2 * time.Hour), UpdatedAt: now.Add(-2 * time.Hour),
	}).Error; err != nil {
		t.Fatal(err)
	}

	h := NewHandler(db, rules.NewLoader("scenarios"))
	r := gin.New()
	h.Register(r, "")

	qReq := httptest.NewRequest(http.MethodGet, "/api/v1/waker/queue?spaceId=local&maxAge=30m", nil)
	qW := httptest.NewRecorder()
	r.ServeHTTP(qW, qReq)
	if qW.Code != http.StatusOK {
		t.Fatalf("queue status=%d body=%s", qW.Code, qW.Body.String())
	}
	var queue waker.QueueResponse
	if err := json.Unmarshal(qW.Body.Bytes(), &queue); err != nil {
		t.Fatal(err)
	}
	if queue.Count < 1 {
		t.Fatalf("queue=%+v", queue)
	}

	sReq := httptest.NewRequest(http.MethodPost, "/api/v1/waker/sweep", bytes.NewReader([]byte(`{"spaceId":"local","dryRun":true,"maxAge":"30m"}`)))
	sReq.Header.Set("Content-Type", "application/json")
	sW := httptest.NewRecorder()
	r.ServeHTTP(sW, sReq)
	if sW.Code != http.StatusOK {
		t.Fatalf("sweep status=%d body=%s", sW.Code, sW.Body.String())
	}
}
