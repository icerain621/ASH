package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ash-repwiki/ash/internal/memory"
	"github.com/ash-repwiki/ash/internal/store"
)

func TestGetMemoryRecordRejectsCrossSpace(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "dev")
	r, db := newPlatformTestRouter(t)
	now := time.Now().UTC()
	mem := store.MemoryRecord{
		ID: "mem_cross_space", Layer: "L1", Status: "approved", SpaceID: "space_other",
		SchemaVersion: memory.CurrentSchemaVersion, Title: "t", Body: "b",
		CreatedAt: now, UpdatedAt: now,
	}
	if err := db.Create(&mem).Error; err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/memory/records/"+mem.ID, nil)
	req.Header.Set("X-ASH-Space-ID", "space_home")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s want 403", w.Code, w.Body.String())
	}
}

func TestGetRunRejectsCrossSpace(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "dev")
	r, db := newPlatformTestRouter(t)
	now := time.Now().UTC()
	run := store.RunRecord{
		ID: "run_cross_space", TraceID: "trace_cross", ScenarioName: "feature_delivery",
		ScenarioVersion: "1.0.0", PolicyProfile: "default", Status: "finished",
		SpaceID: "space_other", StartedAt: now, CreatedAt: now, UpdatedAt: now,
	}
	if err := db.Create(&run).Error; err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/runs/"+run.ID, nil)
	req.Header.Set("X-ASH-Space-ID", "space_home")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s want 403", w.Code, w.Body.String())
	}
}

func TestSpaceMembersRejectsCrossSpaceParam(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "dev")
	r, db := newPlatformTestRouter(t)
	now := time.Now().UTC()
	other := store.Space{
		ID: "space_members_other", OrgID: "org_x", Name: "Other", Slug: "other",
		CreatedAt: now, UpdatedAt: now,
	}
	if err := db.Create(&other).Error; err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/spaces/"+other.ID+"/members", nil)
	req.Header.Set("X-ASH-Space-ID", "space_home")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s want 403", w.Code, w.Body.String())
	}
}

func TestCreateRunRejectsCrossSpaceBody(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "dev")
	r, _ := newPlatformTestRouter(t)
	body, _ := json.Marshal(map[string]any{
		"spaceId": "space_other",
		"scenario": map[string]any{
			"name": "feature_delivery", "scenarioVersion": "1.0.0",
		},
		"inputs": map[string]any{"issueOrSpec": "cross space", "repoRoot": "."},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/runs", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-ASH-Space-ID", "space_home")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s want 403", w.Code, w.Body.String())
	}
}

func TestRotateSecretRejectsCrossSpace(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "dev")
	r, db := newPlatformTestRouter(t)
	now := time.Now().UTC()
	row := store.SecretRecord{
		ID: "sec_cross", SpaceID: "space_other", Name: "TOKEN",
		Status: "active", ScopeJSON: "{}", ValueCiphertext: "x", ValueDigest: "d",
		CreatedAt: now, UpdatedAt: now,
	}
	if err := db.Create(&row).Error; err != nil {
		t.Fatal(err)
	}
	body := []byte(`{"value":"new-secret-value"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/secrets/"+row.ID+"/rotate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-ASH-Space-ID", "space_home")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s want 403", w.Code, w.Body.String())
	}
}
