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

func TestWakerStatusAndDuties(t *testing.T) {
	t.Setenv("ASH_WAKER_INTERVAL", "")
	t.Setenv("ASH_WAKER_RUN_TTL", "30m")
	gin.SetMode(gin.TestMode)
	dir := t.TempDir()
	db, err := store.Open(filepath.Join(dir, "ash.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	h := NewHandler(db, rules.NewLoader("scenarios"))
	r := gin.New()
	h.Register(r, "")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/waker/status?spaceId=local&recent=3", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var st waker.StatusResponse
	if err := json.Unmarshal(w.Body.Bytes(), &st); err != nil {
		t.Fatal(err)
	}
	if len(st.Duties) < 1 || st.Duties[0].Kind != waker.KindStaleRun {
		t.Fatalf("want stale_run duty: %+v", st)
	}
	if st.Interval != "off" || st.IntervalMs != 0 {
		t.Fatalf("want ticker off: interval=%q intervalMs=%d", st.Interval, st.IntervalMs)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/waker/duties?spaceId=local", nil)
	listW := httptest.NewRecorder()
	r.ServeHTTP(listW, listReq)
	if listW.Code != http.StatusOK {
		t.Fatalf("duties status=%d body=%s", listW.Code, listW.Body.String())
	}
	var list struct {
		Duties []waker.DutyStatusView `json:"duties"`
	}
	if err := json.Unmarshal(listW.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if len(list.Duties) < 1 || list.Duties[0].Kind != waker.KindStaleRun {
		t.Fatalf("duties=%+v", list)
	}
}

func TestWakerDutyRun(t *testing.T) {
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
		ID: "run_waker_duty", TraceID: "tr_waker_duty",
		ScenarioName: "feature_delivery", ScenarioVersion: "1.0.0",
		PolicyProfile: "default", Status: runs.StatusRunning, SpaceID: "local",
		RepoRoot: ".", StartedAt: now.Add(-2 * time.Hour),
		CreatedAt: now.Add(-2 * time.Hour), UpdatedAt: now.Add(-2 * time.Hour),
	}).Error; err != nil {
		t.Fatal(err)
	}

	svc := waker.NewService(db)
	duty, err := svc.EnsureStaleRunDuty("local")
	if err != nil {
		t.Fatal(err)
	}

	h := NewHandler(db, rules.NewLoader("scenarios"))
	r := gin.New()
	h.Register(r, "")

	body := bytes.NewReader([]byte(`{"dryRun": true}`))
	runReq := httptest.NewRequest(http.MethodPost, "/api/v1/waker/duties/"+duty.ID+"/run", body)
	runReq.Header.Set("Content-Type", "application/json")
	runW := httptest.NewRecorder()
	r.ServeHTTP(runW, runReq)
	if runW.Code != http.StatusOK {
		t.Fatalf("run status=%d body=%s", runW.Code, runW.Body.String())
	}
	var sweep waker.SweepResponse
	if err := json.Unmarshal(runW.Body.Bytes(), &sweep); err != nil {
		t.Fatal(err)
	}
	if !sweep.DryRun || sweep.Matched < 1 {
		t.Fatalf("sweep=%+v", sweep)
	}
}
