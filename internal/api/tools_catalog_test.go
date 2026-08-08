package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListToolRiskCatalog(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "dev")
	r, _ := newPlatformTestRouter(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tools/risk-catalog", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var resp toolRiskCatalogResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Items) < 5 {
		t.Fatalf("items=%d", len(resp.Items))
	}
	found := false
	for _, item := range resp.Items {
		if item.Name == "runtime.command" && item.Risk == "danger" && item.DefaultDeny {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("missing runtime.command danger entry: %+v", resp.Items)
	}
}
