package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ash-repwiki/ash/internal/agenttpl"
	"github.com/ash-repwiki/ash/internal/plugins"
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
