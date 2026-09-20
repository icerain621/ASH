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

func TestListModelProvidersCatalogMarkers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := store.OpenTest(t, t.TempDir())
	loader := rules.NewLoader(filepath.Join("..", "..", "scenarios"))
	_ = loader.LoadDir()
	h := NewHandler(db, loader)
	r := gin.New()
	h.Register(r, "")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/model-router/providers", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var resp ModelProviderListResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Catalog != "org" || resp.Billing != "none" {
		t.Fatalf("%+v want catalog=org billing=none", resp)
	}
	if resp.Items == nil {
		t.Fatal("items nil")
	}
}
