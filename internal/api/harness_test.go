package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ash-repwiki/ash/internal/harness"
)

func TestHarnessProfileAPILifecycle(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "dev")
	r, _ := newPlatformTestRouter(t)

	body, _ := json.Marshal(harness.CreateRequest{
		Name: "feature-default",
		Spec: harness.DefaultSpec(),
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/harness/profiles", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", w.Code, w.Body.String())
	}
	var created harness.ProfileView
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}

	wRev := httptest.NewRecorder()
	reqRev := httptest.NewRequest(http.MethodPost, "/api/v1/harness/profiles/"+created.ID+"/submit-review", nil)
	r.ServeHTTP(wRev, reqRev)
	if wRev.Code != http.StatusOK {
		t.Fatalf("submit-review status=%d body=%s", wRev.Code, wRev.Body.String())
	}

	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/harness/profiles/"+created.ID+"/promote", nil)
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("promote status=%d body=%s", w2.Code, w2.Body.String())
	}

	w3 := httptest.NewRecorder()
	req3 := httptest.NewRequest(http.MethodGet, "/api/v1/harness/profiles/active?name=feature-default", nil)
	r.ServeHTTP(w3, req3)
	if w3.Code != http.StatusOK {
		t.Fatalf("active status=%d body=%s", w3.Code, w3.Body.String())
	}
	var active harnessLoadActiveResponse
	if err := json.Unmarshal(w3.Body.Bytes(), &active); err != nil {
		t.Fatal(err)
	}
	if active.Profile.ID != created.ID || active.Profile.Status != harness.StatusActive {
		t.Fatalf("active=%+v", active.Profile)
	}
}
