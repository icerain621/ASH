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
		ID        string         `json:"id"`
		RunID     string         `json:"runId"`
		StreamURL string         `json:"streamUrl"`
		Meta      map[string]any `json:"meta"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &sess); err != nil {
		t.Fatal(err)
	}
	if sess.ID == "" || sess.RunID != run.ID || sess.StreamURL == "" {
		t.Fatalf("sess=%+v", sess)
	}
	if tid, _ := sess.Meta["threadId"].(string); tid == "" || sess.Meta["threadKind"] != "main" {
		t.Fatalf("meta=%v want main thread", sess.Meta)
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
			Type       string `json:"type"`
			Visibility string `json:"visibility"`
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
			if item.Visibility != "model_visible" {
				t.Fatalf("session.turn visibility=%q", item.Visibility)
			}
			break
		}
	}
	if !found {
		t.Fatalf("events=%+v want session.turn", evResp.Items)
	}

	// Fail-closed approve while run is running (not waiting_approval).
	actW := httptest.NewRecorder()
	actReq := httptest.NewRequest(http.MethodPost, "/api/v1/agents/sessions/"+sess.ID+"/actions",
		bytes.NewReader([]byte(`{"action":"approve","reason":"nope"}`)))
	actReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(actW, actReq)
	if actW.Code != http.StatusConflict {
		t.Fatalf("approve status=%d body=%s want 409", actW.Code, actW.Body.String())
	}

	promptW := httptest.NewRecorder()
	promptReq := httptest.NewRequest(http.MethodPost, "/api/v1/agents/sessions/"+sess.ID+"/actions",
		bytes.NewReader([]byte(`{"action":"prompt","prompt":"via intent"}`)))
	promptReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(promptW, promptReq)
	if promptW.Code != http.StatusOK {
		t.Fatalf("intent prompt status=%d body=%s", promptW.Code, promptW.Body.String())
	}
}

func TestAgentSessionAPIListBlankAndEvents(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "dev")
	t.Setenv("ASH_AGENT_EXECUTOR", "static")
	r, _ := newPlatformTestRouter(t)

	createW := httptest.NewRecorder()
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/agents/sessions", bytes.NewReader([]byte(`{}`)))
	createReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(createW, createReq)
	if createW.Code != http.StatusCreated {
		t.Fatalf("create blank status=%d body=%s", createW.Code, createW.Body.String())
	}
	var sess struct {
		ID    string `json:"id"`
		RunID string `json:"runId"`
	}
	if err := json.Unmarshal(createW.Body.Bytes(), &sess); err != nil {
		t.Fatal(err)
	}
	if sess.ID == "" || sess.RunID != "" {
		t.Fatalf("sess=%+v want blank", sess)
	}

	listW := httptest.NewRecorder()
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/agents/sessions?limit=50", nil)
	r.ServeHTTP(listW, listReq)
	if listW.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", listW.Code, listW.Body.String())
	}
	var listResp struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	if err := json.Unmarshal(listW.Body.Bytes(), &listResp); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, item := range listResp.Items {
		if item.ID == sess.ID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("list=%+v want %s", listResp.Items, sess.ID)
	}

	turnW := httptest.NewRecorder()
	turnReq := httptest.NewRequest(http.MethodPost, "/api/v1/agents/sessions/"+sess.ID+"/turns",
		bytes.NewReader([]byte(`{"prompt":"blank turn"}`)))
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
		Items []struct {
			Type       string          `json:"type"`
			Seq        int64           `json:"seq"`
			Visibility string          `json:"visibility"`
			Payload    json.RawMessage `json:"payload"`
		} `json:"items"`
	}
	if err := json.Unmarshal(evW.Body.Bytes(), &evResp); err != nil {
		t.Fatal(err)
	}
	if len(evResp.Items) < 2 {
		t.Fatalf("events=%+v want turn + assistant", evResp.Items)
	}
	var sawTurn, sawMsg bool
	for _, item := range evResp.Items {
		switch item.Type {
		case "session.turn":
			sawTurn = true
			if item.Visibility != "model_visible" || item.Seq != 1 {
				t.Fatalf("turn=%+v", item)
			}
			var payload map[string]any
			if err := json.Unmarshal(item.Payload, &payload); err != nil {
				t.Fatal(err)
			}
			if payload["prompt"] != "blank turn" {
				t.Fatalf("payload=%v", payload)
			}
		case "assistant.message":
			sawMsg = true
		}
	}
	if !sawTurn || !sawMsg {
		t.Fatalf("events=%+v want session.turn + assistant.message", evResp.Items)
	}
}

func TestAgentSessionAPIPatchCloseAndStopAlias(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "dev")
	t.Setenv("ASH_AGENT_EXECUTOR", "static")
	r, db := newPlatformTestRouter(t)
	now := time.Now().UTC()
	run := store.RunRecord{
		ID: "run_api_sess_close", TraceID: "trace_api_sess_close",
		ScenarioName: "feature_delivery", ScenarioVersion: "1.0.0",
		PolicyProfile: "default", Status: "running", SpaceID: "local",
		RepoRoot: ".", StartedAt: now, CreatedAt: now, UpdatedAt: now,
	}
	if err := db.Create(&run).Error; err != nil {
		t.Fatal(err)
	}

	createW := httptest.NewRecorder()
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/agents/sessions",
		bytes.NewReader([]byte(`{"runId":"run_api_sess_close"}`)))
	createReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(createW, createReq)
	if createW.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", createW.Code, createW.Body.String())
	}
	var sess struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(createW.Body.Bytes(), &sess); err != nil {
		t.Fatal(err)
	}

	patchW := httptest.NewRecorder()
	patchReq := httptest.NewRequest(http.MethodPatch, "/api/v1/agents/sessions/"+sess.ID,
		bytes.NewReader([]byte(`{"title":"Renamed chat"}`)))
	patchReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(patchW, patchReq)
	if patchW.Code != http.StatusOK {
		t.Fatalf("patch status=%d body=%s", patchW.Code, patchW.Body.String())
	}
	var patched struct {
		Title string `json:"title"`
	}
	if err := json.Unmarshal(patchW.Body.Bytes(), &patched); err != nil {
		t.Fatal(err)
	}
	if patched.Title != "Renamed chat" {
		t.Fatalf("patched=%+v", patched)
	}

	stopW := httptest.NewRecorder()
	stopReq := httptest.NewRequest(http.MethodPost, "/api/v1/agents/sessions/"+sess.ID+"/actions",
		bytes.NewReader([]byte(`{"action":"stop"}`)))
	stopReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(stopW, stopReq)
	if stopW.Code != http.StatusOK {
		t.Fatalf("stop status=%d body=%s", stopW.Code, stopW.Body.String())
	}

	closeW := httptest.NewRecorder()
	closeReq := httptest.NewRequest(http.MethodDelete, "/api/v1/agents/sessions/"+sess.ID, nil)
	r.ServeHTTP(closeW, closeReq)
	if closeW.Code != http.StatusOK {
		t.Fatalf("close status=%d body=%s", closeW.Code, closeW.Body.String())
	}
	var closed struct {
		Status string `json:"status"`
		Title  string `json:"title"`
	}
	if err := json.Unmarshal(closeW.Body.Bytes(), &closed); err != nil {
		t.Fatal(err)
	}
	if closed.Status != "closed" || closed.Title != "Renamed chat" {
		t.Fatalf("closed=%+v", closed)
	}

	listW := httptest.NewRecorder()
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/agents/sessions?limit=50", nil)
	r.ServeHTTP(listW, listReq)
	if listW.Code != http.StatusOK {
		t.Fatalf("list status=%d", listW.Code)
	}
	var listResp struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	if err := json.Unmarshal(listW.Body.Bytes(), &listResp); err != nil {
		t.Fatal(err)
	}
	for _, item := range listResp.Items {
		if item.ID == sess.ID {
			t.Fatalf("closed session visible without includeClosed")
		}
	}

	inclW := httptest.NewRecorder()
	inclReq := httptest.NewRequest(http.MethodGet, "/api/v1/agents/sessions?includeClosed=1", nil)
	r.ServeHTTP(inclW, inclReq)
	if inclW.Code != http.StatusOK {
		t.Fatalf("includeClosed status=%d", inclW.Code)
	}
	if err := json.Unmarshal(inclW.Body.Bytes(), &listResp); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, item := range listResp.Items {
		if item.ID == sess.ID {
			found = true
		}
	}
	if !found {
		t.Fatalf("includeClosed list missing %s", sess.ID)
	}
}
