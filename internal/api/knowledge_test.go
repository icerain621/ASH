package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestKnowledgeProfileAndWiki(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "dev")
	r, _ := newPlatformTestRouter(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/repos/profile?repoRoot=.", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("profile status=%d body=%s", w.Code, w.Body.String())
	}

	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/wiki/pages?repoRoot=.&limit=5", nil)
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("wiki list status=%d body=%s", w2.Code, w2.Body.String())
	}

	w3 := httptest.NewRecorder()
	req3 := httptest.NewRequest(http.MethodGet, "/api/v1/wiki/pages/wiki_profile_overview?repoRoot=.", nil)
	r.ServeHTTP(w3, req3)
	if w3.Code != http.StatusOK {
		t.Fatalf("wiki get status=%d body=%s", w3.Code, w3.Body.String())
	}
}
