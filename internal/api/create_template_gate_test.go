package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/ash-repwiki/ash/internal/agenttpl"
	"github.com/ash-repwiki/ash/internal/plugins"
	"github.com/ash-repwiki/ash/internal/rules"
	"github.com/ash-repwiki/ash/internal/runs"
	"github.com/ash-repwiki/ash/internal/store"
)

func TestCreateRunUnapprovedTemplateForbidden(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "dev")
	agenttpl.ResetCatalogForTest()
	t.Cleanup(agenttpl.ResetCatalogForTest)

	m := agenttpl.Manifest{
		ID: "tpl.custom.api.gate", Version: "1.0.0", Loop: "react",
		Tools: []string{"read"}, Compaction: "threshold", Sandbox: "workspace-write",
		MaxTurns: 2, Memory: []string{"mem.retrieve", "mem.inject"},
	}
	if _, err := agenttpl.SubmitCandidate(m); err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	scenarioYAML := `
version: "ash.rules/v0.1"
scenario:
  name: "custom_tpl_api_gate"
  scenarioVersion: "1.0.0"
  policyProfile: "default"
  roles:
    Coder: { maxParallel: 1 }
  inputs:
    required: [issueOrSpec]
  steps:
    - id: "only"
      role: "Coder"
      kind: "agent"
      agent:
        templateId: "tpl.custom.api.gate"
`
	if err := os.WriteFile(filepath.Join(dir, "custom_tpl_api_gate.yaml"), []byte(scenarioYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	db := store.OpenTest(t, t.TempDir())
	loader := rules.NewLoader(dir)
	if err := loader.LoadDir(); err != nil {
		t.Fatal(err)
	}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	NewHandler(db, loader).Register(r, "")

	body, _ := json.Marshal(runs.CreateRequest{
		Scenario: runs.ScenarioRef{Name: "custom_tpl_api_gate", ScenarioVersion: "1.0.0"},
		Inputs:   map[string]any{"issueOrSpec": "gate"},
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/runs", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var denied APIErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &denied); err != nil {
		t.Fatal(err)
	}
	if denied.Error.Code != "AGENT_TEMPLATE_NOT_APPROVED" {
		t.Fatalf("code=%s", denied.Error.Code)
	}
}

func TestCreateRunInvalidMemoryBindBadRequest(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "dev")
	agenttpl.ResetCatalogForTest()
	t.Cleanup(agenttpl.ResetCatalogForTest)
	agenttpl.ClearApprovedLoaderForTest()
	t.Cleanup(agenttpl.ClearApprovedLoaderForTest)
	plugins.ResetGlobalForTest()
	t.Cleanup(plugins.ResetGlobalForTest)

	dir := t.TempDir()
	scenarioYAML := `
version: "ash.rules/v0.1"
scenario:
  name: "custom_mem_api_gate"
  scenarioVersion: "1.0.0"
  policyProfile: "default"
  roles:
    Coder: { maxParallel: 1 }
  inputs:
    required: [issueOrSpec]
  steps:
    - id: "only"
      role: "Coder"
      kind: "agent"
      agent:
        templateId: "tpl.api.inject.only"
`
	if err := os.WriteFile(filepath.Join(dir, "custom_mem_api_gate.yaml"), []byte(scenarioYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	db := store.OpenTest(t, t.TempDir())
	loader := rules.NewLoader(dir)
	if err := loader.LoadDir(); err != nil {
		t.Fatal(err)
	}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	NewHandler(db, loader).Register(r, "")
	// After Register: handler installs a DB loader; override for this gate case.
	agenttpl.SetApprovedLoader(func(id string) (agenttpl.Manifest, bool) {
		if id != "tpl.api.inject.only" {
			return agenttpl.Manifest{}, false
		}
		return agenttpl.Manifest{
			ID: id, Version: "1.0.0", Loop: "react",
			Tools: []string{"read"}, Compaction: "threshold", Sandbox: "workspace-write",
			MaxTurns: 2, Memory: []string{"mem.inject"},
		}, true
	})

	body, _ := json.Marshal(runs.CreateRequest{
		Scenario: runs.ScenarioRef{Name: "custom_mem_api_gate", ScenarioVersion: "1.0.0"},
		Inputs:   map[string]any{"issueOrSpec": "gate"},
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/runs", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var denied APIErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &denied); err != nil {
		t.Fatal(err)
	}
	if denied.Error.Code != "AGENT_COMPONENT_BIND_INVALID" {
		t.Fatalf("code=%s", denied.Error.Code)
	}
}
