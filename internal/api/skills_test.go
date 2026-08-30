package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListSkillsAPI(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "dev")
	r, _ := newPlatformTestRouter(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/skills?repoRoot=.", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}
