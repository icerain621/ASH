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

func TestGetAgentProviderStatus(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodGet, "/api/v1/providers/agent?spaceId=local", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var body AgentProviderStatus
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Selection.RequestedKind == "" || body.Selection.Adapter == "" {
		t.Fatalf("body=%+v", body)
	}
	if body.ExecGo.Kind != "execgo" {
		t.Fatalf("execGo=%+v", body.ExecGo)
	}
}
