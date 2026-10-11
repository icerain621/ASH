package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ash-repwiki/ash/internal/agenttpl"
	"github.com/ash-repwiki/ash/internal/plugins"
	"github.com/ash-repwiki/ash/internal/review"
	"github.com/ash-repwiki/ash/internal/store"
)

func TestAgentTemplateCandidateAndApproveAPI(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "dev")
	agenttpl.ResetCatalogForTest()
	t.Cleanup(agenttpl.ResetCatalogForTest)
	plugins.ResetGlobalForTest()
	t.Cleanup(plugins.ResetGlobalForTest)

	r, _ := newPlatformTestRouter(t)
	body := []byte(`{"id":"tpl.custom.api","version":"1.0.0","loop":"react","tools":["read"],"compaction":"threshold","sandbox":"workspace-write","maxTurns":2,"memory":["mem.retrieve","mem.inject"]}`)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agent-templates/candidates", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("candidate status=%d body=%s", w.Code, w.Body.String())
	}

	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/agent-templates/tpl.custom.api/approve", nil)
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("approve status=%d body=%s", w2.Code, w2.Body.String())
	}

	w3 := httptest.NewRecorder()
	req3 := httptest.NewRequest(http.MethodPost, "/api/v1/context-pack/preview", bytes.NewReader([]byte(
		`{"issue":"fix","ragRefs":["file:a.go"],"memories":[{"id":"m1","title":"t"}],"memory":["mem.retrieve","mem.inject"]}`,
	)))
	req3.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w3, req3)
	if w3.Code != http.StatusOK {
		t.Fatalf("preview status=%d body=%s", w3.Code, w3.Body.String())
	}
	var pack struct {
		Refs       []string `json:"refs"`
		MemoryRefs []string `json:"memoryRefs"`
	}
	if err := json.Unmarshal(w3.Body.Bytes(), &pack); err != nil {
		t.Fatal(err)
	}
	if len(pack.MemoryRefs) != 1 || pack.Refs[0] != "file:a.go" {
		t.Fatalf("pack=%+v", pack)
	}

	w4 := httptest.NewRecorder()
	req4 := httptest.NewRequest(http.MethodPost, "/api/v1/passk/run", bytes.NewReader([]byte(`{"k":2,"target":"tpl.custom.api"}`)))
	req4.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w4, req4)
	if w4.Code != http.StatusOK {
		t.Fatalf("passk status=%d body=%s", w4.Code, w4.Body.String())
	}
}

func TestMemoryEvalAPI(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "dev")
	r, db := newPlatformTestRouter(t)
	now := time.Now().UTC()
	run := store.RunRecord{
		ID: "run_mem_eval", SpaceID: "local", TraceID: "tr", Status: "succeeded",
		ScenarioName: "x", ScenarioVersion: "1", StartedAt: now, CreatedAt: now, UpdatedAt: now,
	}
	if err := db.Create(&run).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&store.RunEvent{
		ID: "ev_mem_eval", RunID: run.ID, Seq: 1, TS: now.UnixMilli(),
		Type: "memory.feedback", PayloadJSON: `{"kind":"used_correctly","recordId":"m1"}`, CreatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/memory-eval", nil)
	req.Header.Set("X-ASH-Space-ID", "local")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var got review.EvalCounts
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.UsedCorrectly != 1 {
		t.Fatalf("%+v", got)
	}
}

func TestAgentTemplateCandidatePersistsToStore(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "dev")
	agenttpl.ResetCatalogForTest()
	t.Cleanup(agenttpl.ResetCatalogForTest)
	plugins.ResetGlobalForTest()
	t.Cleanup(plugins.ResetGlobalForTest)

	r, db := newPlatformTestRouter(t)
	body := []byte(`{"id":"tpl.custom.db","version":"1.0.0","loop":"react","tools":["read"],"compaction":"threshold","sandbox":"workspace-write","maxTurns":2,"memory":["mem.retrieve","mem.inject"]}`)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agent-templates/candidates", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-ASH-Space-ID", "local")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("candidate status=%d body=%s", w.Code, w.Body.String())
	}
	rows, err := db.ListAgentTemplateVersions("local")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, row := range rows {
		if row.TemplateID == "tpl.custom.db" && row.Status == "candidate" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected durable candidate, rows=%+v", rows)
	}
	pending, err := db.ListPendingReviewItems("local", "agent_template")
	if err != nil || len(pending) < 1 {
		t.Fatalf("pending=%+v err=%v", pending, err)
	}
}

func TestListAgentReviewsAPI(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "dev")
	agenttpl.ResetCatalogForTest()
	t.Cleanup(agenttpl.ResetCatalogForTest)
	plugins.ResetGlobalForTest()
	t.Cleanup(plugins.ResetGlobalForTest)

	r, _ := newPlatformTestRouter(t)
	body := []byte(`{"id":"tpl.custom.rev","version":"1.0.0","loop":"react","tools":["read"],"compaction":"threshold","sandbox":"workspace-write","maxTurns":2,"memory":["mem.retrieve","mem.inject"]}`)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agent-templates/candidates", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("candidate status=%d body=%s", w.Code, w.Body.String())
	}

	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/agent-reviews?kind=agent_template", nil)
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", w2.Code, w2.Body.String())
	}
	var listed struct {
		Items []struct {
			Kind     string `json:"kind"`
			TargetID string `json:"targetId"`
			Status   string `json:"status"`
		} `json:"items"`
	}
	if err := json.Unmarshal(w2.Body.Bytes(), &listed); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, item := range listed.Items {
		if item.TargetID == "tpl.custom.rev" && item.Kind == "agent_template" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("listed=%+v", listed.Items)
	}

	w3 := httptest.NewRecorder()
	req3 := httptest.NewRequest(http.MethodGet, "/api/v1/agent-templates", nil)
	r.ServeHTTP(w3, req3)
	if w3.Code != http.StatusOK {
		t.Fatalf("templates status=%d", w3.Code)
	}
	var entries struct {
		Items []struct {
			Status   string `json:"status"`
			Manifest struct {
				ID string `json:"id"`
			} `json:"manifest"`
		} `json:"items"`
	}
	if err := json.Unmarshal(w3.Body.Bytes(), &entries); err != nil {
		t.Fatal(err)
	}
	var customStatus string
	for _, item := range entries.Items {
		if item.Manifest.ID == "tpl.custom.rev" {
			customStatus = item.Status
		}
	}
	if customStatus != "candidate" {
		t.Fatalf("want candidate status, items=%+v", entries.Items)
	}
}

func TestPluginComponentRegisterApproveAPI(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "dev")
	plugins.ResetGlobalForTest()
	t.Cleanup(plugins.ResetGlobalForTest)

	r, _ := newPlatformTestRouter(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agent-components", bytes.NewReader([]byte(
		`{"id":"tool.custom.api","kind":"tool","status":"candidate"}`,
	)))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("register status=%d body=%s", w.Code, w.Body.String())
	}
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/agent-components/tool.custom.api/approve", nil)
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("approve status=%d body=%s", w2.Code, w2.Body.String())
	}
	w3 := httptest.NewRecorder()
	req3 := httptest.NewRequest(http.MethodGet, "/api/v1/agent-components", nil)
	r.ServeHTTP(w3, req3)
	if w3.Code != http.StatusOK {
		t.Fatalf("list status=%d", w3.Code)
	}
}
