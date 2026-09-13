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

func TestInteractionByRunFoldAndLinks(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "dev")
	t.Setenv("ASH_AGENT_EXECUTOR", "static")
	r, db := newPlatformTestRouter(t)
	now := time.Now().UTC()
	run := store.RunRecord{
		ID: "run_ix_1", TraceID: "tr_ix_1",
		ScenarioName: "feature_delivery", ScenarioVersion: "1.0.0",
		PolicyProfile: "default", Status: "running", SpaceID: "local",
		RepoRoot: ".", StartedAt: now, CreatedAt: now, UpdatedAt: now,
	}
	if err := db.Create(&run).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&store.RunEvent{
		ID: "evt_ix_1", RunID: run.ID, Seq: 1, TS: now.UnixMilli(),
		Type: "memory.hit_used", Severity: "info", Visibility: "model_visible",
		PayloadJSON: `{"recordIds":["mem_ix"],"count":1}`, CreatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/interactions/by-run/"+run.ID, nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("by-run status=%d body=%s", w.Code, w.Body.String())
	}
	var byRun struct {
		Thread struct {
			ID    string `json:"id"`
			RunID string `json:"runId"`
		} `json:"thread"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &byRun); err != nil {
		t.Fatal(err)
	}
	if byRun.Thread.ID == "" || byRun.Thread.RunID != run.ID {
		t.Fatalf("byRun=%+v", byRun)
	}

	tw := httptest.NewRecorder()
	treq := httptest.NewRequest(http.MethodGet, "/api/v1/interactions/threads/"+byRun.Thread.ID, nil)
	r.ServeHTTP(tw, treq)
	if tw.Code != http.StatusOK {
		t.Fatalf("fold status=%d body=%s", tw.Code, tw.Body.String())
	}
	var fold struct {
		Digest string `json:"digest"`
		Links  []struct {
			MemoryID string `json:"memoryId"`
			LinkType string `json:"linkType"`
		} `json:"links"`
	}
	if err := json.Unmarshal(tw.Body.Bytes(), &fold); err != nil {
		t.Fatal(err)
	}
	if fold.Digest == "" || len(fold.Links) != 1 || fold.Links[0].MemoryID != "mem_ix" {
		t.Fatalf("fold=%+v", fold)
	}

	lw := httptest.NewRecorder()
	lreq := httptest.NewRequest(http.MethodGet, "/api/v1/interactions/threads/"+byRun.Thread.ID+"/memory-links", nil)
	r.ServeHTTP(lw, lreq)
	if lw.Code != http.StatusOK {
		t.Fatalf("links status=%d body=%s", lw.Code, lw.Body.String())
	}

	ew := httptest.NewRecorder()
	ereq := httptest.NewRequest(http.MethodPost, "/api/v1/interactions/threads/ensure",
		bytes.NewReader([]byte(`{"runId":"run_ix_1"}`)))
	ereq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(ew, ereq)
	if ew.Code != http.StatusOK {
		t.Fatalf("ensure status=%d body=%s", ew.Code, ew.Body.String())
	}

	sw := httptest.NewRecorder()
	sreq := httptest.NewRequest(http.MethodPost, "/api/v1/interactions/threads/"+byRun.Thread.ID+"/seal", nil)
	r.ServeHTTP(sw, sreq)
	if sw.Code != http.StatusOK {
		t.Fatalf("seal status=%d body=%s", sw.Code, sw.Body.String())
	}
	rw := httptest.NewRecorder()
	rreq := httptest.NewRequest(http.MethodPost, "/api/v1/interactions/threads/"+byRun.Thread.ID+"/replay", nil)
	r.ServeHTTP(rw, rreq)
	if rw.Code != http.StatusOK {
		t.Fatalf("replay status=%d body=%s", rw.Code, rw.Body.String())
	}
}

func TestListInteractionSessionThreads(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "dev")
	t.Setenv("ASH_AGENT_EXECUTOR", "static")
	r, db := newPlatformTestRouter(t)
	now := time.Now().UTC()
	for _, runID := range []string{"run_sess_th_a", "run_sess_th_b"} {
		run := store.RunRecord{
			ID: runID, TraceID: "tr_" + runID,
			ScenarioName: "feature_delivery", ScenarioVersion: "1.0.0",
			PolicyProfile: "default", Status: "running", SpaceID: "local",
			RepoRoot: ".", StartedAt: now, CreatedAt: now, UpdatedAt: now,
		}
		if err := db.Create(&run).Error; err != nil {
			t.Fatal(err)
		}
		ew := httptest.NewRecorder()
		ereq := httptest.NewRequest(http.MethodPost, "/api/v1/interactions/threads/ensure",
			bytes.NewReader([]byte(`{"runId":"`+runID+`","sessionId":"sess_multi"}`)))
		ereq.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(ew, ereq)
		if ew.Code != http.StatusOK {
			t.Fatalf("ensure %s status=%d body=%s", runID, ew.Code, ew.Body.String())
		}
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/interactions/sessions/sess_multi/threads", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("list threads status=%d body=%s", w.Code, w.Body.String())
	}
	var out struct {
		SessionID string `json:"sessionId"`
		Items     []struct {
			ID    string `json:"id"`
			RunID string `json:"runId"`
		} `json:"items"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.SessionID != "sess_multi" || len(out.Items) != 2 {
		t.Fatalf("out=%+v", out)
	}
}
