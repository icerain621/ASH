package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ash-repwiki/ash/internal/runs"
	"github.com/ash-repwiki/ash/internal/store"
)

func TestSubRunTreeAndDepthReject(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "dev")
	r, db := newPlatformTestRouter(t)
	now := time.Now().UTC()
	parent := store.RunRecord{
		ID: "run_do_parent", TraceID: "trc_do", ScenarioName: "feature_delivery", ScenarioVersion: "1.0.0",
		PolicyProfile: "default", Status: "finished", SpaceID: "local", ActorRole: "maintainer",
		Depth: 0, StartedAt: now, CreatedAt: now, UpdatedAt: now,
	}
	if err := db.Create(&parent).Error; err != nil {
		t.Fatal(err)
	}

	wTree := httptest.NewRecorder()
	r.ServeHTTP(wTree, httptest.NewRequest(http.MethodGet, "/api/v1/runs/run_do_parent/tree", nil))
	if wTree.Code != http.StatusOK {
		t.Fatalf("tree status=%d body=%s", wTree.Code, wTree.Body.String())
	}
	var tree runs.TreeResponse
	if err := json.Unmarshal(wTree.Body.Bytes(), &tree); err != nil {
		t.Fatal(err)
	}
	if tree.RootRunID != "run_do_parent" || tree.Tree.Summary.RunID != "run_do_parent" {
		t.Fatalf("%+v", tree)
	}

	deep := store.RunRecord{
		ID: "run_do_deep", TraceID: "trc_deep", ScenarioName: "feature_delivery", ScenarioVersion: "1.0.0",
		PolicyProfile: "default", Status: "finished", SpaceID: "local", ActorRole: "maintainer",
		Depth: 2, StartedAt: now, CreatedAt: now, UpdatedAt: now,
	}
	if err := db.Create(&deep).Error; err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(runs.SpawnRequest{
		Scenario: runs.ScenarioRef{Name: "hotfix", ScenarioVersion: "1.0.0"},
		Inputs:   map[string]any{"issueOrSpec": "x", "repoRoot": "."},
	})
	wSpawn := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/runs/run_do_deep/sub-runs", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(wSpawn, req)
	if wSpawn.Code == http.StatusCreated {
		t.Fatalf("expected depth reject, got 201 body=%s", wSpawn.Body.String())
	}
}
