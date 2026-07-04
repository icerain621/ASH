//go:build integration

package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/ash-repwiki/ash/internal/rules"
	"github.com/ash-repwiki/ash/internal/store"
)

func TestPostgresReadyzProbe(t *testing.T) {
	pgURL := os.Getenv("ASH_DATABASE_URL")
	if pgURL == "" {
		t.Skip("ASH_DATABASE_URL unset")
	}
	if os.Getenv("ASH_MIGRATE_E2E") != "1" {
		t.Skip("set ASH_MIGRATE_E2E=1 for live postgres readyz probe")
	}
	if err := store.ResetPostgresPublicSchema(pgURL); err != nil {
		t.Fatalf("reset postgres: %v", err)
	}

	dir := t.TempDir()
	db, err := store.OpenWithDatabaseURL(dir, pgURL)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if db.Dialect() != "postgres" {
		t.Fatalf("dialect=%q want postgres", db.Dialect())
	}

	gin.SetMode(gin.TestMode)
	loader := rules.NewLoader(resolveIntegrationScenariosDir())
	if err := loader.LoadDir(); err != nil {
		t.Fatal(err)
	}
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
	if resp.Status != "ready" || resp.Dialect != "postgres" {
		t.Fatalf("readyz=%+v want ready/postgres", resp)
	}
	if resp.SQLMigrationExpected != 20 {
		t.Fatalf("sqlMigrationExpected=%d want 20", resp.SQLMigrationExpected)
	}
}

func TestPostgresReadyzWithRLS(t *testing.T) {
	pgURL := os.Getenv("ASH_DATABASE_URL")
	if pgURL == "" {
		t.Skip("ASH_DATABASE_URL unset")
	}
	t.Setenv("ASH_POSTGRES_RLS", "1")
	t.Setenv("ASH_POSTGRES_RLS_FORCE", "1")

	dir := t.TempDir()
	db, err := store.OpenWithDatabaseURL(dir, pgURL)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if db.Dialect() != "postgres" {
		t.Fatalf("dialect=%q want postgres", db.Dialect())
	}

	gin.SetMode(gin.TestMode)
	loader := rules.NewLoader(resolveIntegrationScenariosDir())
	if err := loader.LoadDir(); err != nil {
		t.Fatal(err)
	}
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
	if !resp.PostgresRLSEnabled {
		t.Fatalf("readyz=%+v want postgresRLSEnabled", resp)
	}
	want := int64(store.PostgresRLSExpectedPolicyCount())
	if resp.PostgresRLSPolicyCount < want {
		t.Fatalf("postgresRLSPolicyCount=%d want >= %d", resp.PostgresRLSPolicyCount, want)
	}
	if resp.PostgresRLSPolicyExpected != want {
		t.Fatalf("postgresRLSPolicyExpected=%d want %d", resp.PostgresRLSPolicyExpected, want)
	}
	if resp.RLSCatalogSummary == "" {
		t.Fatal("expected rlsCatalogSummary on postgres readyz with RLS")
	}
}

func TestPostgresReadyzScaleParity(t *testing.T) {
	pgURL := os.Getenv("ASH_DATABASE_URL")
	if pgURL == "" {
		t.Skip("ASH_DATABASE_URL unset")
	}
	t.Setenv("ASH_AUTH_MODE", "dev")
	t.Setenv("ASH_POSTGRES_RLS", "1")
	t.Setenv("ASH_POSTGRES_RLS_FORCE", "1")

	dir := t.TempDir()
	db, err := store.OpenWithDatabaseURL(dir, pgURL)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	gin.SetMode(gin.TestMode)
	loader := rules.NewLoader(resolveIntegrationScenariosDir())
	if err := loader.LoadDir(); err != nil {
		t.Fatal(err)
	}
	h := NewHandler(db, loader)
	r := gin.New()
	h.Register(r, "")

	wz := httptest.NewRecorder()
	r.ServeHTTP(wz, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if wz.Code != http.StatusOK {
		t.Fatalf("readyz status=%d body=%s", wz.Code, wz.Body.String())
	}
	var readyz HealthResponse
	if err := json.Unmarshal(wz.Body.Bytes(), &readyz); err != nil {
		t.Fatal(err)
	}

	ws := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/scale/readiness", nil)
	req.Header.Set("X-ASH-Space-ID", "local")
	r.ServeHTTP(ws, req)
	if ws.Code != http.StatusOK {
		t.Fatalf("scale status=%d body=%s", ws.Code, ws.Body.String())
	}
	var scale ScaleReadinessResponse
	if err := json.Unmarshal(ws.Body.Bytes(), &scale); err != nil {
		t.Fatal(err)
	}
	if err := AssertReadyzScaleParity(readyz, scale); err != nil {
		t.Fatal(err)
	}
}

func resolveIntegrationScenariosDir() string {
	for _, p := range []string{
		filepath.Join("..", "..", "scenarios"),
		"scenarios",
	} {
		if st, err := os.Stat(p); err == nil && st.IsDir() {
			return p
		}
	}
	return "scenarios"
}
