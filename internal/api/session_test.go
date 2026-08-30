package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ash-repwiki/ash/internal/store"
)

func TestAgentSessionAPIBindRunTurnEvents(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "dev")
	t.Setenv("ASH_AGENT_EXECUTOR", "static")
	r, db := newPlatformTestRouter(t)
	now := time.Now().UTC()
	run := store.RunRecord{
		ID: "run_api_sess", TraceID: "trace_api_sess",
		ScenarioName: "feature_delivery", ScenarioVersion: "1.0.0",
		PolicyProfile: "default", Status: "running", SpaceID: "local",
		RepoRoot: ".", StartedAt: now, CreatedAt: now, UpdatedAt: now,
	}
	if err := db.Create(&run).Error; err != nil {
		t.Fatal(err)
	}

	body := []byte(`{"runId":"run_api_sess"}`)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents/sessions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", w.Code, w.Body.String())
	}
	var sess struct {
		ID        string `json:"id"`
		RunID     string `json:"runId"`
		StreamURL string `json:"streamUrl"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &sess); err != nil {
		t.Fatal(err)
	}
	if sess.ID == "" || sess.RunID != run.ID || sess.StreamURL == "" {
		t.Fatalf("sess=%+v", sess)
	}

	turnW := httptest.NewRecorder()
	turnReq := httptest.NewRequest(http.MethodPost, "/api/v1/agents/sessions/"+sess.ID+"/turns",
		bytes.NewReader([]byte(`{"prompt":"please continue"}`)))
	turnReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(turnW, turnReq)
	if turnW.Code != http.StatusOK {
		t.Fatalf("turn status=%d body=%s", turnW.Code, turnW.Body.String())
	}

	evW := httptest.NewRecorder()
	evReq := httptest.NewRequest(http.MethodGet, "/api/v1/agents/sessions/"+sess.ID+"/events", nil)
	r.ServeHTTP(evW, evReq)
	if evW.Code != http.StatusOK {
		t.Fatalf("events status=%d body=%s", evW.Code, evW.Body.String())
	}
	var evResp struct {
		StreamURL string `json:"streamUrl"`
		Items     []struct {
			Type string `json:"type"`
		} `json:"items"`
	}
	if err := json.Unmarshal(evW.Body.Bytes(), &evResp); err != nil {
		t.Fatal(err)
	}
	if evResp.StreamURL == "" {
		t.Fatalf("evResp=%+v", evResp)
	}
	found := false
	for _, item := range evResp.Items {
		if item.Type == "session.turn" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("events=%+v want session.turn", evResp.Items)
	}
}
