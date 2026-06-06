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
