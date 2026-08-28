package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ash-repwiki/ash/internal/evolve"
	"github.com/ash-repwiki/ash/internal/harness"
)

func TestReviewsQueueHarnessDecide(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "dev")
	r, db := newPlatformTestRouter(t)
	har := harness.NewService(db)
	created, err := har.Create(harness.CreateRequest{Name: "orch", Spec: harness.DefaultSpec()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := har.SubmitReview(created.ID); err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/reviews/queue?queue=orchestration", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("queue status=%d body=%s", w.Code, w.Body.String())
	}
	var queue reviewsQueueResponse
	if err := json.Unmarshal(w.Body.Bytes(), &queue); err != nil {
		t.Fatal(err)
	}
	if len(queue.Items) < 1 {
		t.Fatal("expected orchestration item")
	}

	body, _ := json.Marshal(evolve.DecideRequest{Decision: "approve", Reason: "ok"})
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/reviews/"+queue.Items[0].ID+"/decide", bytes.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("decide status=%d body=%s", w2.Code, w2.Body.String())
	}

	w3 := httptest.NewRecorder()
	fb, _ := json.Marshal(map[string]any{
		"targetType": "harness_profile", "targetId": created.ID, "rating": 2, "runId": "run_dy_1",
	})
	req3 := httptest.NewRequest(http.MethodPost, "/api/v1/feedback", bytes.NewReader(fb))
	req3.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w3, req3)
	if w3.Code != http.StatusCreated {
		t.Fatalf("feedback status=%d body=%s", w3.Code, w3.Body.String())
	}
}
