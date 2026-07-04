package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/ash-repwiki/ash/internal/rules"
	"github.com/ash-repwiki/ash/internal/store"
)

func TestReadyzOpsSnapshot(t *testing.T) {
	t.Setenv("ASH_METRICS_EVENT_REPLAY", "1")
	t.Setenv("ASH_ALERTS_EVAL_INTERVAL", "5m")
	t.Setenv("ASH_MEMORY_TTL_SWEEP_INTERVAL", "24h")

	gin.SetMode(gin.TestMode)
	db := store.OpenTest(t, t.TempDir())
	loader := rules.NewLoader(filepath.Join("..", "..", "scenarios"))
	_ = loader.LoadDir()
	h := NewHandler(db, loader)
	r := gin.New()
	h.Register(r, "")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("readyz status=%d body=%s", w.Code, w.Body.String())
	}
	var resp HealthResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if !resp.MetricsEventReplayEnabled {
		t.Fatal("expected metricsEventReplayEnabled")
	}
	if resp.AlertsEvalInterval != "5m0s" {
		t.Fatalf("alertsEvalInterval=%q want 5m0s", resp.AlertsEvalInterval)
	}
	if resp.MemoryTTLSweepInterval != "24h0m0s" {
		t.Fatalf("memoryTTLSweepInterval=%q want 24h0m0s", resp.MemoryTTLSweepInterval)
	}
	if resp.Dialect != "sqlite" {
		t.Fatalf("dialect=%q want sqlite", resp.Dialect)
	}
}

func TestReadyzLiveGateHints(t *testing.T) {
	t.Setenv("ASH_MIGRATE_E2E", "1")
	t.Setenv("ASH_POSTGRES_RLS", "1")

	gin.SetMode(gin.TestMode)
	db := store.OpenTest(t, t.TempDir())
	loader := rules.NewLoader(filepath.Join("..", "..", "scenarios"))
	_ = loader.LoadDir()
	h := NewHandler(db, loader)
	r := gin.New()
	h.Register(r, "")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("readyz status=%d body=%s", w.Code, w.Body.String())
	}
	var resp HealthResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(resp.LiveGateHints, "\n")
	if !strings.Contains(joined, "ASH_MIGRATE_E2E=1") || !strings.Contains(joined, "ASH_POSTGRES_RLS=1") {
		t.Fatalf("liveGateHints=%v want migrate and rls gates", resp.LiveGateHints)
	}
}
