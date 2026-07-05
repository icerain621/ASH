package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ash-repwiki/ash/internal/rules"
	"github.com/ash-repwiki/ash/internal/store"
)

func TestHealthzAndReadyzSQLite(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := store.OpenTest(t, t.TempDir())
	loader := rules.NewLoader(filepath.Join("..", "..", "scenarios"))
	_ = loader.LoadDir()
	h := NewHandler(db, loader)
	r := gin.New()
	h.Register(r, "")

	for _, path := range []string{"/healthz", "/readyz"} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != http.StatusOK {
			t.Fatalf("%s status=%d body=%s", path, w.Code, w.Body.String())
		}
		var resp HealthResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}
		if path == "/readyz" {
			if resp.Status != "ready" || resp.Dialect != "sqlite" {
				t.Fatalf("readyz=%+v want ready/sqlite", resp)
			}
			if resp.OtelEnabled || resp.MetricsEventReplayEnabled || resp.AlertsEvalInterval != "" {
				t.Fatalf("readyz=%+v want default ops flags unset", resp)
			}
		} else if resp.Status != "ok" {
			t.Fatalf("healthz=%+v want ok", resp)
		}
	}
}

const healthBaselineMaxP95 = 250 * time.Millisecond

func TestHealthEndpointsLatencyBaseline(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := store.OpenTest(t, t.TempDir())
	loader := rules.NewLoader(filepath.Join("..", "..", "scenarios"))
	_ = loader.LoadDir()
	h := NewHandler(db, loader)
	r := gin.New()
	h.Register(r, "")

	latencies := make([]time.Duration, 0, 40)
	for i := 0; i < 40; i++ {
		for _, path := range []string{"/healthz", "/readyz"} {
			start := time.Now()
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
			latencies = append(latencies, time.Since(start))
			if w.Code != http.StatusOK {
				t.Fatalf("%s status=%d", path, w.Code)
			}
		}
	}
	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
	p95 := latencies[(len(latencies)*95)/100]
	if p95 > healthBaselineMaxP95 {
		t.Fatalf("p95=%s want <= %s", p95, healthBaselineMaxP95)
	}
}

func TestConcurrentRunsListBaseline(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "dev")
	gin.SetMode(gin.TestMode)
	db := store.OpenTest(t, t.TempDir())
	loader := rules.NewLoader(filepath.Join("..", "..", "scenarios"))
	_ = loader.LoadDir()
	h := NewHandler(db, loader)
	r := gin.New()
	h.Register(r, "")

	const workers = 12
	errCh := make(chan error, workers)
	start := time.Now()
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/runs", nil)
			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				errCh <- &baselineHTTPError{code: w.Code, body: w.Body.String()}
			}
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Fatal(err)
	}
	elapsed := time.Since(start)
	if elapsed > 2*time.Second {
		t.Fatalf("12 concurrent /runs took %s want < 2s", elapsed)
	}
}

type baselineHTTPError struct {
	code int
	body string
}

func (e *baselineHTTPError) Error() string {
	return fmt.Sprintf("status=%d body=%s", e.code, e.body)
}
