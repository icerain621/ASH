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
		} else if resp.Status != "ok" {
			t.Fatalf("healthz=%+v want ok", resp)
		}
	}
}
