package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/ash-repwiki/ash/internal/rules"
	"github.com/ash-repwiki/ash/internal/store"
)

func TestReadyzIncludesRLSCatalogWhenEnabled(t *testing.T) {
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
	if !resp.PostgresRLSEnabled {
		t.Log("postgresRLSEnabled false on sqlite (RLS inactive until dialect=postgres)")
	}
	want := int64(store.PostgresRLSExpectedPolicyCount())
	if resp.PostgresRLSPolicyExpected != want {
		t.Fatalf("postgresRLSPolicyExpected=%d want %d", resp.PostgresRLSPolicyExpected, want)
	}
	if resp.RLSCatalogSummary == "" {
		t.Fatal("expected rlsCatalogSummary on readyz when RLS enabled")
	}
	if resp.SQLMigrationExpected == 0 {
		t.Fatal("expected sqlMigrationExpected on readyz")
	}
}
